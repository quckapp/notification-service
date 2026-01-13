import { Injectable, Logger, NotFoundException } from '@nestjs/common';
import { InjectRepository } from '@nestjs/typeorm';
import { Repository } from 'typeorm';
import { NotificationTemplate } from './entities/template.entity';
import { NotificationType } from '../notification/entities/notification.entity';

export class CreateTemplateDto {
  name: string;
  description?: string;
  type: NotificationType;
  subject: string;
  bodyTemplate: string;
  htmlTemplate?: string;
  defaultData?: Record<string, any>;
  category?: string;
  workspaceId?: string;
}

export class UpdateTemplateDto {
  name?: string;
  description?: string;
  subject?: string;
  bodyTemplate?: string;
  htmlTemplate?: string;
  defaultData?: Record<string, any>;
  category?: string;
  isActive?: boolean;
}

@Injectable()
export class TemplateService {
  private readonly logger = new Logger(TemplateService.name);

  constructor(
    @InjectRepository(NotificationTemplate)
    private readonly templateRepo: Repository<NotificationTemplate>,
  ) {}

  async create(dto: CreateTemplateDto): Promise<NotificationTemplate> {
    const template = this.templateRepo.create(dto);
    await this.templateRepo.save(template);
    this.logger.log(`Created template: ${template.name}`);
    return template;
  }

  async findAll(workspaceId?: string): Promise<NotificationTemplate[]> {
    const query = this.templateRepo.createQueryBuilder('t');
    if (workspaceId) {
      query.where('t.workspaceId = :workspaceId OR t.workspaceId IS NULL', {
        workspaceId,
      });
    }
    return query.orderBy('t.name', 'ASC').getMany();
  }

  async findById(id: string): Promise<NotificationTemplate> {
    const template = await this.templateRepo.findOne({ where: { id } });
    if (!template) {
      throw new NotFoundException('Template not found');
    }
    return template;
  }

  async findByName(name: string): Promise<NotificationTemplate> {
    const template = await this.templateRepo.findOne({ where: { name } });
    if (!template) {
      throw new NotFoundException('Template not found');
    }
    return template;
  }

  async update(id: string, dto: UpdateTemplateDto): Promise<NotificationTemplate> {
    const template = await this.findById(id);
    Object.assign(template, dto);
    await this.templateRepo.save(template);
    this.logger.log(`Updated template: ${template.name}`);
    return template;
  }

  async delete(id: string): Promise<void> {
    const template = await this.findById(id);
    await this.templateRepo.remove(template);
    this.logger.log(`Deleted template: ${template.name}`);
  }

  renderTemplate(
    template: NotificationTemplate,
    data: Record<string, any>,
  ): { subject: string; body: string; html?: string } {
    const mergedData = { ...template.defaultData, ...data };

    const subject = this.interpolate(template.subject, mergedData);
    const body = this.interpolate(template.bodyTemplate, mergedData);
    const html = template.htmlTemplate
      ? this.interpolate(template.htmlTemplate, mergedData)
      : undefined;

    return { subject, body, html };
  }

  private interpolate(template: string, data: Record<string, any>): string {
    return template.replace(/\{\{(\w+)\}\}/g, (match, key) => {
      return data[key] !== undefined ? String(data[key]) : match;
    });
  }
}
