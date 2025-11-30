package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

// RabbitMQ RabbitMQ封装
type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQ 创建RabbitMQ实例
func NewRabbitMQ(url string) (*RabbitMQ, error) {
	// 连接RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// 创建通道
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &RabbitMQ{
		conn:    conn,
		channel: channel,
	}, nil
}

// DeclareQueue 声明队列
func (mq *RabbitMQ) DeclareQueue(queueName string) error {
	_, err := mq.channel.QueueDeclare(
		queueName, // 队列名称
		true,      // durable：持久化
		false,     // autoDelete：自动删除
		false,     // exclusive：独占
		false,     // noWait：不等待
		nil,       // arguments：额外参数
	)
	return err
}

// Publish 发布消息
func (mq *RabbitMQ) Publish(queueName string, message interface{}) error {
	// 1. 声明队列（确保队列存在）
	if err := mq.DeclareQueue(queueName); err != nil {
		return err
	}

	// 2. 序列化消息
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// 3. 发布消息
	err = mq.channel.Publish(
		"",        // exchange：默认交换机
		queueName, // routing key：队列名称
		false,     // mandatory：强制
		false,     // immediate：立即
		amqp.Publishing{
			DeliveryMode: amqp.Persistent, // 持久化消息
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)

	return err
}

// Consume 消费消息
func (mq *RabbitMQ) Consume(queueName string, handler func([]byte) error) error {
	// 1. 声明队列
	if err := mq.DeclareQueue(queueName); err != nil {
		return err
	}

	// 2. 设置QoS（每次只处理1条消息）
	if err := mq.channel.Qos(1, 0, false); err != nil {
		return err
	}

	// 3. 开始消费
	msgs, err := mq.channel.Consume(
		queueName, // 队列名称
		"",        // consumer：消费者标识
		false,     // autoAck：手动确认
		false,     // exclusive：独占
		false,     // noLocal：不接收同一连接的消息
		false,     // noWait：不等待
		nil,       // arguments：额外参数
	)
	if err != nil {
		return err
	}

	// 4. 处理消息
	go func() {
		for msg := range msgs {
			// 调用处理函数
			if err := handler(msg.Body); err != nil {
				// 处理失败，拒绝消息并重新入队
				msg.Nack(false, true)
			} else {
				// 处理成功，确认消息
				msg.Ack(false)
			}
		}
	}()

	return nil
}

// ConsumeWithContext 带上下文的消费（支持优雅关闭）
func (mq *RabbitMQ) ConsumeWithContext(ctx context.Context, queueName string, handler func([]byte) error) error {
	// 1. 声明队列
	if err := mq.DeclareQueue(queueName); err != nil {
		return err
	}

	// 2. 设置QoS
	if err := mq.channel.Qos(1, 0, false); err != nil {
		return err
	}

	// 3. 开始消费
	msgs, err := mq.channel.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// 4. 处理消息（支持上下文取消）
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				// 调用处理函数
				if err := handler(msg.Body); err != nil {
					msg.Nack(false, true)
				} else {
					msg.Ack(false)
				}
			}
		}
	}()

	return nil
}

// PublishWithRetry 发布消息（带重试）
func (mq *RabbitMQ) PublishWithRetry(queueName string, message interface{}, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = mq.Publish(queueName, message)
		if err == nil {
			return nil
		}

		// 指数退避
		time.Sleep(time.Duration(1<<i) * time.Second)
	}
	return fmt.Errorf("failed to publish after %d retries: %w", maxRetries, err)
}

// Close 关闭连接
func (mq *RabbitMQ) Close() error {
	if err := mq.channel.Close(); err != nil {
		return err
	}
	return mq.conn.Close()
}

// GetChannel 获取原始通道（用于高级操作）
func (mq *RabbitMQ) GetChannel() *amqp.Channel {
	return mq.channel
}
