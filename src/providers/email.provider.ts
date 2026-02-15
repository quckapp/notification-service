import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import * as nodemailer from 'nodemailer';
import type { Transporter } from 'nodemailer';

export interface EmailPayload {
  to: string;
  subject: string;
  text?: string;
  html?: string;
  from?: string;
  replyTo?: string;
  attachments?: Array<{
    filename: string;
    content: Buffer | string;
    contentType?: string;
  }>;
}

export interface EmailResult {
  success: boolean;
  messageId?: string;
  error?: string;
}

@Injectable()
export class EmailProvider implements OnModuleInit {
  private readonly logger = new Logger(EmailProvider.name);
  private transporter: Transporter | null = null;
  private defaultFrom: string;

  constructor(private readonly configService: ConfigService) {}

  onModuleInit() {
    this.initialize();
  }

  private initialize(): void {
    const host = this.configService.get<string>('SMTP_HOST');
    const port = this.configService.get<number>('SMTP_PORT', 587);
    const user = this.configService.get<string>('SMTP_USER');
    const password = this.configService.get<string>('SMTP_PASSWORD');
    this.defaultFrom = this.configService.get<string>('SMTP_FROM', 'noreply@quckapp.com');

    if (!host || !user || !password) {
      this.logger.warn(
        'SMTP credentials not configured. Email notifications will be disabled.',
      );
      return;
    }

    try {
      this.transporter = nodemailer.createTransport({
        host,
        port,
        secure: port === 465,
        auth: {
          user,
          pass: password,
        },
        pool: true,
        maxConnections: 5,
        maxMessages: 100,
        rateDelta: 1000,
        rateLimit: 10,
      });

      // Verify connection
      this.transporter.verify((error) => {
        if (error) {
          this.logger.error('SMTP connection verification failed:', error);
          this.transporter = null;
        } else {
          this.logger.log('SMTP connection established successfully');
        }
      });
    } catch (error) {
      this.logger.error('Failed to initialize email transporter:', error);
    }
  }

  isInitialized(): boolean {
    return this.transporter !== null;
  }

  async sendEmail(payload: EmailPayload): Promise<EmailResult> {
    if (!this.transporter) {
      this.logger.warn('Email transporter not initialized, skipping email');
      return {
        success: false,
        error: 'Email transporter not initialized',
      };
    }

    try {
      const mailOptions = {
        from: payload.from || this.defaultFrom,
        to: payload.to,
        subject: payload.subject,
        text: payload.text,
        html: payload.html,
        replyTo: payload.replyTo,
        attachments: payload.attachments,
      };

      const info = await this.transporter.sendMail(mailOptions);

      this.logger.debug(`Email sent to ${payload.to}: ${info.messageId}`);

      return {
        success: true,
        messageId: info.messageId,
      };
    } catch (error) {
      this.logger.error(`Failed to send email to ${payload.to}: ${error.message}`);
      return {
        success: false,
        error: error.message,
      };
    }
  }

  async sendBulkEmails(payloads: EmailPayload[]): Promise<EmailResult[]> {
    const results = await Promise.all(
      payloads.map((payload) => this.sendEmail(payload)),
    );

    const successCount = results.filter((r) => r.success).length;
    this.logger.log(`Bulk email sent: ${successCount}/${payloads.length} successful`);

    return results;
  }

  async sendTemplatedEmail(
    to: string,
    subject: string,
    templateHtml: string,
    variables: Record<string, string>,
  ): Promise<EmailResult> {
    // Simple variable interpolation
    let html = templateHtml;
    for (const [key, value] of Object.entries(variables)) {
      html = html.replace(new RegExp(`{{${key}}}`, 'g'), value);
    }

    // Generate plain text from HTML
    const text = html
      .replace(/<[^>]*>/g, '')
      .replace(/\s+/g, ' ')
      .trim();

    return this.sendEmail({
      to,
      subject,
      html,
      text,
    });
  }
}
