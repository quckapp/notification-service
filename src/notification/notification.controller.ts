import {
  Controller,
  Get,
  Post,
  Body,
  Param,
  Query,
  Headers,
  ParseUUIDPipe,
} from '@nestjs/common';
import { ApiTags, ApiOperation, ApiResponse, ApiHeader } from '@nestjs/swagger';
import { NotificationService } from './notification.service';
import {
  SendNotificationDto,
  SendBulkNotificationDto,
  MarkReadDto,
  NotificationResponseDto,
} from './dto/notification.dto';

@ApiTags('notifications')
@Controller('notifications')
export class NotificationController {
  constructor(private readonly notificationService: NotificationService) {}

  @Post()
  @ApiOperation({ summary: 'Send a notification' })
  @ApiResponse({ status: 201, type: NotificationResponseDto })
  async send(@Body() dto: SendNotificationDto): Promise<NotificationResponseDto> {
    return this.notificationService.send(dto);
  }

  @Post('bulk')
  @ApiOperation({ summary: 'Send bulk notifications' })
  async sendBulk(@Body() dto: SendBulkNotificationDto): Promise<{ queued: number }> {
    return this.notificationService.sendBulk(dto);
  }

  @Get('user/:userId')
  @ApiOperation({ summary: 'Get user notifications' })
  async getUserNotifications(
    @Param('userId', ParseUUIDPipe) userId: string,
    @Query('workspaceId') workspaceId?: string,
    @Query('page') page?: number,
    @Query('limit') limit?: number,
  ): Promise<{ data: NotificationResponseDto[]; total: number }> {
    return this.notificationService.getUserNotifications(
      userId,
      workspaceId,
      page || 0,
      limit || 20,
    );
  }

  @Get('user/:userId/unread-count')
  @ApiOperation({ summary: 'Get unread notification count' })
  async getUnreadCount(
    @Param('userId', ParseUUIDPipe) userId: string,
    @Query('workspaceId') workspaceId?: string,
  ): Promise<{ count: number }> {
    const count = await this.notificationService.getUnreadCount(userId, workspaceId);
    return { count };
  }

  @Post('read')
  @ApiOperation({ summary: 'Mark notifications as read' })
  @ApiHeader({ name: 'X-User-Id', required: true })
  async markAsRead(
    @Body() dto: MarkReadDto,
    @Headers('X-User-Id') userId: string,
  ): Promise<{ success: boolean }> {
    await this.notificationService.markAsRead(userId, dto.notificationIds);
    return { success: true };
  }

  @Post('read-all')
  @ApiOperation({ summary: 'Mark all notifications as read' })
  @ApiHeader({ name: 'X-User-Id', required: true })
  async markAllAsRead(
    @Headers('X-User-Id') userId: string,
    @Query('workspaceId') workspaceId?: string,
  ): Promise<{ success: boolean }> {
    await this.notificationService.markAllAsRead(userId, workspaceId);
    return { success: true };
  }
}
