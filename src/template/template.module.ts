import { Module } from '@nestjs/common';
import { TypeOrmModule } from '@nestjs/typeorm';
import { NotificationTemplate } from './entities/template.entity';
import { TemplateService } from './template.service';
import { TemplateController } from './template.controller';

@Module({
  imports: [TypeOrmModule.forFeature([NotificationTemplate])],
  controllers: [TemplateController],
  providers: [TemplateService],
  exports: [TemplateService],
})
export class TemplateModule {}
