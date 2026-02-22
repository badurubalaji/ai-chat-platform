import { ComponentFixture, TestBed } from '@angular/core/testing';

import { MdpAiChat } from './mdp-ai-chat';

describe('MdpAiChat', () => {
  let component: MdpAiChat;
  let fixture: ComponentFixture<MdpAiChat>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [MdpAiChat]
    })
    .compileComponents();

    fixture = TestBed.createComponent(MdpAiChat);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
