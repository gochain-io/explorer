import {
  AfterViewInit,
  Directive,
  ElementRef,
  EventEmitter,
  Input,
  NgZone,
  OnChanges,
  Output,
  TemplateRef,
  ViewContainerRef
} from '@angular/core';
import {fromEvent, Subscription} from 'rxjs';
import {debounceTime, distinctUntilChanged, filter} from 'rxjs/operators';
import {AutoUnsubscribe} from '../decorators/auto-unsubscribe';

@Directive({
  selector: '[appInfinityScroll]'
})
@AutoUnsubscribe('_subsArr$')
export class InfinityScrollDirective implements OnChanges, AfterViewInit {
  @Input('appInfinityScroll') active: boolean;
  @Output() onView = new EventEmitter<void>();
  debounceInterval = 100;
  private _subsArr$: Subscription[] = [];
  private _target: any;

  constructor(private _viewContainer: ViewContainerRef, private _templateRef: TemplateRef<any>, private _zone: NgZone) {
  }

  ngAfterViewInit() {
    this._target = document.getElementsByClassName('app-content')[0];
    this.initTracker();
  }

  ngOnChanges() {
    if (this.active) {
      this._viewContainer.createEmbeddedView(this._templateRef);
    } else {
      this._viewContainer.clear();
    }
  }

  initTracker() {
    this._subsArr$.push(this._zone.runOutsideAngular(() =>
      fromEvent(this._target, 'scroll').pipe(
        debounceTime(this.debounceInterval),
        distinctUntilChanged(),
        filter(() => {
          const targetTop = this._templateRef.elementRef.nativeElement.nextSibling.offsetTop;
          const containerBottom = this._target.scrollTop + this._target.offsetHeight + 300;
          return containerBottom > targetTop;
        }),
      ).subscribe((e: any) => {
        e.preventDefault();
        this._zone.run(() => {
          this.onView.emit();
        });
      })
    ));
  }
}
