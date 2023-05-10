package main

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"log"
	"os"
	"strings"
)

var userMessages = make(map[string][]openai.ChatCompletionMessage)

func main() {
	godotenv.Load()
	// Discord bot tokeni buraya yazın
	discordToken := os.Getenv("DISCORD_TOKEN")

	// Discord botu oluşturma
	dg, err := discordgo.New("Bot " + discordToken)
	if err != nil {
		log.Fatal("Error creating Discord session: ", err)
	}

	// Bot hazır olduğunda çağrılacak fonksiyonu ayarla
	dg.AddHandler(onReady)

	// Mesajlar dinleniyor
	dg.AddHandler(onMessage)

	// Discord API'ye bağlan
	err = dg.Open()
	if err != nil {
		log.Fatal("Error opening connection to Discord: ", err)
	}

	// Botu çalışır durumda tut
	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	<-make(chan struct{})
	return
}

func onReady(session *discordgo.Session, event *discordgo.Ready) {
	// Bot hazır olduğunda log kaydı
	fmt.Println("Bot is now connected to Discord.")
}

func onMessage(session *discordgo.Session, message *discordgo.MessageCreate) {
	// Mesajı gönderen kullanıcının bot olup olmadığını kontrol et
	if message.Author.Bot {
		return
	}

	// Mesajın içeriğini al
	content := message.Content
	userID := message.Author.ID

	if content == "/gpt -clear" {
		delete(userMessages, userID)
		// Yanıtı mesaj olarak gönder
		session.ChannelMessageSend(message.ChannelID, "Önbellek temizlendi.")
		return
	}

	if _, ok := userMessages[userID]; !ok {
		userMessages[userID] = []openai.ChatCompletionMessage{}
	}

	userMessages[userID] = append(userMessages[userID], openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: content,
	})

	// Mesajın başında "chatgpt" var mı diye kontrol et
	if strings.HasPrefix(content, "/gpt") {

		content = strings.Replace(content, "/gpt ", "", 1)

		// Yanıtı mesaj olarak gönder
		lastMsg, err := session.ChannelMessageSend(message.ChannelID, "Araştırılıyor...")

		// Chatgpt istek
		client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model:    openai.GPT3Dot5Turbo,
				Messages: userMessages[userID],
			},
		)

		if err != nil {
			fmt.Printf("ChatCompletion error: %v\n", err)
			return
		}

		// Botun yanıtını geçmişe ekle
		userMessages[userID] = append(userMessages[userID], openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: resp.Choices[0].Message.Content,
		})

		err = session.ChannelMessageDelete(message.ChannelID, lastMsg.ID)
		// Yanıtı mesaj olarak gönder
		session.ChannelMessageSend(message.ChannelID, resp.Choices[0].Message.Content)
	}
}
