package service

import (
	"bufio"
	"fmt"
	"github.com/geziang/popim"
	"io"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

const (
	dataPath = "./data/"
	avatarPath = dataPath + "avatars/"
	tempPath = dataPath + "temp/"
	avatarSuffix = ".jpg"
	defaultAvatarPath = avatarPath+"default"+avatarSuffix //默认头像路径
)

const (
	fileBlockSize = 1024 //文件分块大小
)

var (
	FileService = &fileService{}
	bytePool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, fileBlockSize)
			return &b
		},
	}
)

type fileService struct {}

/*
 流式分块发送头像文件
 */
func (*fileService) SendAvatarFile(popId uint, stream popim.User_GetAvatarServer) {
	//生成文件路径
	filePath := fmt.Sprint(avatarPath,popId,avatarSuffix)
	info, err := os.Stat(filePath)
	if err != nil {
		filePath = defaultAvatarPath
		info, _ = os.Stat(filePath)
	}
	fileSize := info.Size()
	totalBlocks := fileSize / fileBlockSize
	if fileSize %fileBlockSize != 0 {
		totalBlocks ++
	}
	log.Println("Sending file ", filePath, " size=", fileSize, " total_blocks=", totalBlocks)

	//准备资源
	file, err := os.Open(filePath)
	if err != nil {
		log.Println("Open file ", filePath, " fail err:",err)
		return
	}
	reader := bufio.NewReader(file)
	defer file.Close()
	buf := bytePool.Get().(*[]byte)
	defer bytePool.Put(buf)

	//开始传输
	for{
		n, err := reader.Read(*buf)
		if err != nil{
			if err != io.EOF {
				log.Println("File read error ", err, "file=", filePath)
			}
			break
		}

		err = stream.Send(&popim.StreamDownloadFileBlock{
			TotalCount: uint64(totalBlocks),
			Data:       (*buf)[:n],
		})
		if err != nil{
			log.Println("File block send error ", err, "file=", filePath)
			break
		}
	}
}

/*
 流式分块接收头像文件
*/
func (svc *fileService) RecvAvatarFile(stream popim.User_UpdateAvatarServer) {
	//生成文件路径
	filePath := fmt.Sprint(tempPath,time.Now().Unix(),"_",rand.Int(),avatarSuffix)
	targetPath := ""

	//准备资源
	file, err := os.Create(filePath)
	defer os.Remove(filePath)
	if err != nil {
		log.Println("Create file ", filePath, " fail err:",err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	//开始传输
	for{
		block, err := stream.Recv()
		if err != nil{
			if err == io.EOF {
				writer.Flush()
				file.Seek(0,0) //回到文件开头以便读取
				log.Println("File recved ", filePath)
				if svc.copyFile(file, targetPath) {
					log.Println("File saved ", targetPath)
				}
			} else {
				log.Println("File block recv error ", err, "file=", filePath)
			}
			break
		}

		if targetPath == "" {
			//先认证
			popId := uint(block.Context.PopId)
			token := block.Context.Token

			success, err := UserService.CheckToken(popId,token)
			if err != nil {
				break
			}
			if !success {
				//认证失败
				log.Println("Avatar Upload auth failed!")
				break
			}

			//生成文件路径
			targetPath = fmt.Sprint(avatarPath,popId,avatarSuffix)
			log.Println("Recving file ", filePath, "target file ", targetPath, " total_blocks=", block.TotalCount)
		}

		_, err = writer.Write(block.Data)
		if err != nil{
			log.Println("File write error ", err, "file=", filePath)
			break
		}
	}
}

/*
 拷贝文件
 */
func (*fileService) copyFile(srcFile *os.File, targetPath string) bool {
	//准备资源
	file, err := os.Create(targetPath)
	if err != nil {
		log.Println("Create file ", targetPath, " fail err:",err)
		return false
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	_, err = io.Copy(writer, srcFile)
	if err != nil {
		log.Println("Copy file to ", targetPath, " fail err:",err)
		return false
	}
	return true
}