import {
  Controller,
  Get,
  Post,
  Put,
  Delete,
  Body,
  Param,
  Query,
  ParseUUIDPipe,
} from '@nestjs/common';
import { ApiTags, ApiOperation } from '@nestjs/swagger';
import {
  TemplateService,
  CreateTemplateDto,
  UpdateTemplateDto,
} from './template.service';
import { NotificationTemplate } from './entities/template.entity';

@ApiTags('templates')
@Controller('templates')
export class TemplateController {
  constructor(private readonly templateService: TemplateService) {}

  @Post()
  @ApiOperation({ summary: 'Create notification template' })
  async create(@Body() dto: CreateTemplateDto): Promise<NotificationTemplate> {
    return this.templateService.create(dto);
  }

  @Get()
  @ApiOperation({ summary: 'Get all templates' })
  async findAll(
    @Query('workspaceId') workspaceId?: string,
  ): Promise<NotificationTemplate[]> {
    return this.templateService.findAll(workspaceId);
  }

  @Get(':id')
  @ApiOperation({ summary: 'Get template by ID' })
  async findById(
    @Param('id', ParseUUIDPipe) id: string,
  ): Promise<NotificationTemplate> {
    return this.templateService.findById(id);
  }

  @Get('name/:name')
  @ApiOperation({ summary: 'Get template by name' })
  async findByName(@Param('name') name: string): Promise<NotificationTemplate> {
    return this.templateService.findByName(name);
  }

  @Put(':id')
  @ApiOperation({ summary: 'Update template' })
  async update(
    @Param('id', ParseUUIDPipe) id: string,
    @Body() dto: UpdateTemplateDto,
  ): Promise<NotificationTemplate> {
    return this.templateService.update(id, dto);
  }

  @Delete(':id')
  @ApiOperation({ summary: 'Delete template' })
  async delete(
    @Param('id', ParseUUIDPipe) id: string,
  ): Promise<{ success: boolean }> {
    await this.templateService.delete(id);
    return { success: true };
  }

  @Post(':id/render')
  @ApiOperation({ summary: 'Render template with data' })
  async render(
    @Param('id', ParseUUIDPipe) id: string,
    @Body() data: Record<string, any>,
  ): Promise<{ subject: string; body: string; html?: string }> {
    const template = await this.templateService.findById(id);
    return this.templateService.renderTemplate(template, data);
  }
}
