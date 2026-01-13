import {
  Controller,
  Get,
  Put,
  Post,
  Delete,
  Body,
  Param,
  Query,
  ParseUUIDPipe,
} from '@nestjs/common';
import { ApiTags, ApiOperation } from '@nestjs/swagger';
import { PreferenceService, UpdatePreferenceDto } from './preference.service';
import { NotificationPreference } from './entities/preference.entity';

@ApiTags('preferences')
@Controller('preferences')
export class PreferenceController {
  constructor(private readonly preferenceService: PreferenceService) {}

  @Get(':userId')
  @ApiOperation({ summary: 'Get user notification preferences' })
  async getPreferences(
    @Param('userId', ParseUUIDPipe) userId: string,
    @Query('workspaceId') workspaceId?: string,
  ): Promise<NotificationPreference> {
    return this.preferenceService.getOrCreatePreferences(userId, workspaceId);
  }

  @Put(':userId')
  @ApiOperation({ summary: 'Update notification preferences' })
  async updatePreferences(
    @Param('userId', ParseUUIDPipe) userId: string,
    @Query('workspaceId') workspaceId: string | null,
    @Body() dto: UpdatePreferenceDto,
  ): Promise<NotificationPreference> {
    return this.preferenceService.updatePreferences(userId, workspaceId, dto);
  }

  @Post(':userId/mute-channel/:channelId')
  @ApiOperation({ summary: 'Mute a channel' })
  async muteChannel(
    @Param('userId', ParseUUIDPipe) userId: string,
    @Param('channelId', ParseUUIDPipe) channelId: string,
    @Query('workspaceId', ParseUUIDPipe) workspaceId: string,
  ): Promise<{ success: boolean }> {
    await this.preferenceService.muteChannel(userId, workspaceId, channelId);
    return { success: true };
  }

  @Delete(':userId/mute-channel/:channelId')
  @ApiOperation({ summary: 'Unmute a channel' })
  async unmuteChannel(
    @Param('userId', ParseUUIDPipe) userId: string,
    @Param('channelId', ParseUUIDPipe) channelId: string,
    @Query('workspaceId', ParseUUIDPipe) workspaceId: string,
  ): Promise<{ success: boolean }> {
    await this.preferenceService.unmuteChannel(userId, workspaceId, channelId);
    return { success: true };
  }
}
