import { ComponentFixture, TestBed } from '@angular/core/testing';

import { AiUsageDashboard } from './ai-usage-dashboard';

describe('AiUsageDashboard', () => {
  let component: AiUsageDashboard;
  let fixture: ComponentFixture<AiUsageDashboard>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [AiUsageDashboard]
    })
    .compileComponents();

    fixture = TestBed.createComponent(AiUsageDashboard);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
