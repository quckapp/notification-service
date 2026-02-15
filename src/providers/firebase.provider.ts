import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as admin from 'firebase-admin';

export interface PushPayload {
  title: string;
  body: string;
  data?: Record<string, string>;
  imageUrl?: string;
  badge?: number;
  sound?: string;
  priority?: 'high' | 'normal';
}

export interface PushResult {
  success: boolean;
  successCount: number;
  failureCount: number;
  failedTokens: string[];
  errors: Array<{ token: string; error: string }>;
}

@Injectable()
export class FirebaseProvider implements OnModuleInit {
  private readonly logger = new Logger(FirebaseProvider.name);
  private initialized = false;

  constructor(private readonly configService: ConfigService) {}

  onModuleInit() {
    this.initialize();
  }

  private initialize(): void {
    const projectId = this.configService.get<string>('FIREBASE_PROJECT_ID');
    const privateKey = this.configService.get<string>('FIREBASE_PRIVATE_KEY');
    const clientEmail = this.configService.get<string>('FIREBASE_CLIENT_EMAIL');

    if (!projectId || !privateKey || !clientEmail) {
      this.logger.warn(
        'Firebase credentials not configured. Push notifications will be disabled.',
      );
      return;
    }

    try {
      // Check if already initialized
      if (admin.apps.length === 0) {
        admin.initializeApp({
          credential: admin.credential.cert({
            projectId,
            privateKey: privateKey.replace(/\\n/g, '\n'),
            clientEmail,
          }),
        });
      }
      this.initialized = true;
      this.logger.log('Firebase Admin SDK initialized successfully');
    } catch (error) {
      this.logger.error('Failed to initialize Firebase Admin SDK:', error);
    }
  }

  isInitialized(): boolean {
    return this.initialized;
  }

  async sendToDevice(token: string, payload: PushPayload): Promise<PushResult> {
    if (!this.initialized) {
      this.logger.warn('Firebase not initialized, skipping push notification');
      return {
        success: false,
        successCount: 0,
        failureCount: 1,
        failedTokens: [token],
        errors: [{ token, error: 'Firebase not initialized' }],
      };
    }

    try {
      const message: admin.messaging.Message = {
        token,
        notification: {
          title: payload.title,
          body: payload.body,
          imageUrl: payload.imageUrl,
        },
        data: payload.data,
        android: {
          priority: payload.priority === 'high' ? 'high' : 'normal',
          notification: {
            sound: payload.sound || 'default',
            channelId: 'default',
          },
        },
        apns: {
          payload: {
            aps: {
              badge: payload.badge,
              sound: payload.sound || 'default',
            },
          },
        },
      };

      await admin.messaging().send(message);
      this.logger.debug(`Push notification sent to token: ${token.substring(0, 20)}...`);

      return {
        success: true,
        successCount: 1,
        failureCount: 0,
        failedTokens: [],
        errors: [],
      };
    } catch (error) {
      this.logger.error(`Failed to send push notification: ${error.message}`);
      return {
        success: false,
        successCount: 0,
        failureCount: 1,
        failedTokens: [token],
        errors: [{ token, error: error.message }],
      };
    }
  }

  async sendToDevices(tokens: string[], payload: PushPayload): Promise<PushResult> {
    if (!this.initialized) {
      this.logger.warn('Firebase not initialized, skipping push notifications');
      return {
        success: false,
        successCount: 0,
        failureCount: tokens.length,
        failedTokens: tokens,
        errors: tokens.map((token) => ({ token, error: 'Firebase not initialized' })),
      };
    }

    if (tokens.length === 0) {
      return {
        success: true,
        successCount: 0,
        failureCount: 0,
        failedTokens: [],
        errors: [],
      };
    }

    try {
      const message: admin.messaging.MulticastMessage = {
        tokens,
        notification: {
          title: payload.title,
          body: payload.body,
          imageUrl: payload.imageUrl,
        },
        data: payload.data,
        android: {
          priority: payload.priority === 'high' ? 'high' : 'normal',
          notification: {
            sound: payload.sound || 'default',
            channelId: 'default',
          },
        },
        apns: {
          payload: {
            aps: {
              badge: payload.badge,
              sound: payload.sound || 'default',
            },
          },
        },
      };

      const response = await admin.messaging().sendEachForMulticast(message);

      const failedTokens: string[] = [];
      const errors: Array<{ token: string; error: string }> = [];

      response.responses.forEach((resp, idx) => {
        if (!resp.success) {
          failedTokens.push(tokens[idx]);
          errors.push({
            token: tokens[idx],
            error: resp.error?.message || 'Unknown error',
          });
        }
      });

      this.logger.log(
        `Push notifications sent: ${response.successCount} success, ${response.failureCount} failed`,
      );

      return {
        success: response.failureCount === 0,
        successCount: response.successCount,
        failureCount: response.failureCount,
        failedTokens,
        errors,
      };
    } catch (error) {
      this.logger.error(`Failed to send multicast push notification: ${error.message}`);
      return {
        success: false,
        successCount: 0,
        failureCount: tokens.length,
        failedTokens: tokens,
        errors: tokens.map((token) => ({ token, error: error.message })),
      };
    }
  }

  async subscribeToTopic(tokens: string[], topic: string): Promise<void> {
    if (!this.initialized) {
      this.logger.warn('Firebase not initialized, skipping topic subscription');
      return;
    }

    try {
      await admin.messaging().subscribeToTopic(tokens, topic);
      this.logger.log(`Subscribed ${tokens.length} devices to topic: ${topic}`);
    } catch (error) {
      this.logger.error(`Failed to subscribe to topic ${topic}: ${error.message}`);
      throw error;
    }
  }

  async unsubscribeFromTopic(tokens: string[], topic: string): Promise<void> {
    if (!this.initialized) {
      this.logger.warn('Firebase not initialized, skipping topic unsubscription');
      return;
    }

    try {
      await admin.messaging().unsubscribeFromTopic(tokens, topic);
      this.logger.log(`Unsubscribed ${tokens.length} devices from topic: ${topic}`);
    } catch (error) {
      this.logger.error(`Failed to unsubscribe from topic ${topic}: ${error.message}`);
      throw error;
    }
  }

  async sendToTopic(topic: string, payload: PushPayload): Promise<boolean> {
    if (!this.initialized) {
      this.logger.warn('Firebase not initialized, skipping topic notification');
      return false;
    }

    try {
      const message: admin.messaging.Message = {
        topic,
        notification: {
          title: payload.title,
          body: payload.body,
          imageUrl: payload.imageUrl,
        },
        data: payload.data,
      };

      await admin.messaging().send(message);
      this.logger.log(`Push notification sent to topic: ${topic}`);
      return true;
    } catch (error) {
      this.logger.error(`Failed to send to topic ${topic}: ${error.message}`);
      return false;
    }
  }
}
