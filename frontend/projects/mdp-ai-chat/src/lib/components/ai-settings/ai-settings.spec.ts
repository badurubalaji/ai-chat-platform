import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AiSettings } from './ai-settings';

describe('AiSettings', () => {
  let component: AiSettings;
  let fixture: ComponentFixture<AiSettings>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AiSettings]
    })
    .compileComponents();

    fixture = TestBed.createComponent(AiSettings);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
