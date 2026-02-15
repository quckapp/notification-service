import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as twilio from 'twilio';
import type { Twilio } from 'twilio';
import type { MessageInstance } from 'twilio/lib/rest/api/v2010/account/message';

export interface SmsPayload {
  to: string;
  body: string;
  from?: string;
  mediaUrl?: string[];
}

export interface SmsResult {
  success: boolean;
  messageId?: string;
  status?: string;
  error?: string;
}

@Injectable()
export class SmsProvider implements OnModuleInit {
  private readonly logger = new Logger(SmsProvider.name);
  private client: Twilio | null = null;
  private defaultFrom: string;

  constructor(private readonly configService: ConfigService) {}

  onModuleInit() {
    this.initialize();
  }

  private initialize(): void {
    const accountSid = this.configService.get<string>('TWILIO_ACCOUNT_SID');
    const authToken = this.configService.get<string>('TWILIO_AUTH_TOKEN');
    this.defaultFrom = this.configService.get<string>('TWILIO_PHONE_NUMBER', '');

    if (!accountSid || !authToken) {
      this.logger.warn(
        'Twilio credentials not configured. SMS notifications will be disabled.',
      );
      return;
    }

    try {
      this.client = twilio(accountSid, authToken);
      this.logger.log('Twilio client initialized successfully');
    } catch (error) {
      this.logger.error('Failed to initialize Twilio client:', error);
    }
  }

  isInitialized(): boolean {
    return this.client !== null;
  }

  async sendSms(payload: SmsPayload): Promise<SmsResult> {
    if (!this.client) {
      this.logger.warn('Twilio client not initialized, skipping SMS');
      return {
        success: false,
        error: 'Twilio client not initialized',
      };
    }

    if (!this.defaultFrom && !payload.from) {
      return {
        success: false,
        error: 'No sender phone number configured',
      };
    }

    try {
      // Format phone number if needed
      const toNumber = this.formatPhoneNumber(payload.to);

      const messageOptions: {
        to: string;
        from: string;
        body: string;
        mediaUrl?: string[];
      } = {
        to: toNumber,
        from: payload.from || this.defaultFrom,
        body: payload.body,
      };

      if (payload.mediaUrl && payload.mediaUrl.length > 0) {
        messageOptions.mediaUrl = payload.mediaUrl;
      }

      const message = await this.client.messages.create(messageOptions);

      this.logger.debug(`SMS sent to ${toNumber}: ${message.sid}`);

      return {
        success: true,
        messageId: message.sid,
        status: message.status,
      };
    } catch (error) {
      this.logger.error(`Failed to send SMS to ${payload.to}: ${error.message}`);
      return {
        success: false,
        error: error.message,
      };
    }
  }

  async sendBulkSms(payloads: SmsPayload[]): Promise<SmsResult[]> {
    const results = await Promise.all(
      payloads.map((payload) => this.sendSms(payload)),
    );

    const successCount = results.filter((r) => r.success).length;
    this.logger.log(`Bulk SMS sent: ${successCount}/${payloads.length} successful`);

    return results;
  }

  async getMessageStatus(messageId: string): Promise<string | null> {
    if (!this.client) {
      return null;
    }

    try {
      const message = await this.client.messages(messageId).fetch();
      return message.status;
    } catch (error) {
      this.logger.error(`Failed to fetch message status: ${error.message}`);
      return null;
    }
  }

  private formatPhoneNumber(phone: string): string {
    // Remove any non-digit characters except +
    let formatted = phone.replace(/[^\d+]/g, '');

    // Add + prefix if not present and number starts with country code
    if (!formatted.startsWith('+')) {
      // Assume it needs a + prefix
      formatted = '+' + formatted;
    }

    return formatted;
  }

  // Estimate SMS segments (for cost estimation)
  getSegmentCount(body: string): number {
    const length = body.length;
    const hasUnicode = /[^\x00-\x7F]/.test(body);

    if (hasUnicode) {
      // Unicode SMS: 70 chars per segment, 67 for multipart
      return length <= 70 ? 1 : Math.ceil(length / 67);
    } else {
      // GSM-7 SMS: 160 chars per segment, 153 for multipart
      return length <= 160 ? 1 : Math.ceil(length / 153);
    }
  }
}
