import { TestBed } from '@angular/core/testing';
import { HttpClientTestingModule, HttpTestingController } from '@angular/common/http/testing';
import { AiChatService } from './ai-chat.service';
import { AI_CHAT_CONFIG, AiChatConfig } from '../providers/ai-chat-config';
import { of } from 'rxjs';

describe('AiChatService', () => {
    let service: AiChatService;
    let httpMock: HttpTestingController;

    const mockConfig: AiChatConfig = {
        apiBaseUrl: 'http://localhost:8080',
        authTokenFn: () => of('mock-token')
    };

    beforeEach(() => {
        TestBed.configureTestingModule({
            imports: [HttpClientTestingModule],
            providers: [
                AiChatService,
                { provide: AI_CHAT_CONFIG, useValue: mockConfig }
            ]
        });
        service = TestBed.inject(AiChatService);
        httpMock = TestBed.inject(HttpTestingController);
    });

    afterEach(() => {
        httpMock.verify();
    });

    it('should be created', () => {
        expect(service).toBeTruthy();
    });

    it('should list conversations', () => {
        const mockConversations = [
            { id: '1', title: 'Test Chat', created_at: '2024-01-01' }
        ];

        service.getConversations().subscribe(convos => {
            expect(convos.length).toBe(1);
            expect(convos[0].title).toBe('Test Chat');
        });

        const req = httpMock.expectOne('http://localhost:8080/api/v1/ai/conversations');
        expect(req.request.method).toBe('GET');
        req.flush(mockConversations);
    });

    it('should create conversation', () => {
        const newConvo = { id: '2', title: 'New Chat', created_at: '2024-01-02' };

        service.createConversation().subscribe(convo => {
            expect(convo.title).toBe('New Chat');
        });

        const req = httpMock.expectOne('http://localhost:8080/api/v1/ai/conversations');
        expect(req.request.method).toBe('POST');
        req.flush(newConvo);
    });
});
