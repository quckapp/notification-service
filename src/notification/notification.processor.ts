import { Process, Processor } from '@nestjs/bull';
import { Logger } from '@nestjs/common';
import { Job } from 'bull';
import { NotificationService } from './notification.service';

@Processor('notifications')
export class NotificationProcessor {
  private readonly logger = new Logger(NotificationProcessor.name);

  constructor(private readonly notificationService: NotificationService) {}

  @Process('send')
  async handleSend(job: Job<{ notificationId: string }>) {
    this.logger.debug(`Processing notification: ${job.data.notificationId}`);
    await this.notificationService.processNotification(job.data.notificationId);
    this.logger.debug(`Completed notification: ${job.data.notificationId}`);
  }
}
