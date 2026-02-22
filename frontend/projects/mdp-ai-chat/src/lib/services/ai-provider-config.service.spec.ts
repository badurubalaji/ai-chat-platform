import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { AiProviderConfigService } from './ai-provider-config.service';
import { AI_CHAT_CONFIG, AiChatConfig } from '../providers/ai-chat-config';
import { AiProviderConfig } from '../models/ai-chat.model';
import { of } from 'rxjs';

describe('AiProviderConfigService', () => {
    let service: AiProviderConfigService;
    let httpMock: HttpTestingController;

    const mockConfig: AiChatConfig = {
        apiBaseUrl: 'http://localhost:8080',
        authTokenFn: () => of('mock-token')
    };

    beforeEach(() => {
        TestBed.configureTestingModule({
            imports: [HttpClientTestingModule],
            providers: [
                AiProviderConfigService,
                { provide: AI_CHAT_CONFIG, useValue: mockConfig }
            ]
        });
        service = TestBed.inject(AiProviderConfigService);
        httpMock = TestBed.inject(HttpTestingController);
    });

    afterEach(() => {
        httpMock.verify();
    });

    it('should be created', () => {
        expect(service).toBeTruthy();
    });

    it('should get config', () => {
        const mockResponse: AiProviderConfig = {
            id: '123',
            tenant_id: 'default',
            provider: 'openai',
            model: 'gpt-4',
            settings: { temperature: 0.7, max_tokens: 100 },
            enabled: true
        };

        service.getConfig().subscribe(config => {
            expect(config).toEqual(mockResponse);
        });

        const req = httpMock.expectOne('http://localhost:8080/api/v1/ai/config');
        expect(req.request.method).toBe('GET');
        req.flush(mockResponse);
    });

    it('should save config', () => {
        const newConfig: AiProviderConfig = {
            id: '',
            tenant_id: 'default',
            provider: 'claude',
            model: 'claude-3-opus',
            apiKey: 'sk-test',
            settings: { temperature: 0.5, max_tokens: 200 },
            enabled: true
        };

        const savedConfig = { ...newConfig, id: '456' };

        service.saveConfig(newConfig).subscribe(config => {
            expect(config).toEqual(savedConfig);
        });

        const req = httpMock.expectOne('http://localhost:8080/api/v1/ai/config');
        expect(req.request.method).toBe('POST');
        expect(req.request.body).toEqual(newConfig);
        req.flush(savedConfig);
    });
});
