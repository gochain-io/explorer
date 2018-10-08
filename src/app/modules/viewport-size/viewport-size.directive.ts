import {Directive, Input, OnDestroy, OnInit, TemplateRef, ViewContainerRef} from '@angular/core';
import {ViewportSizeService} from './viewport-size.service';
import {ViewportSizeEnum} from './viewport-size.enum';
import {filter} from 'rxjs/operators';


@Directive({selector: '[ifViewportSize]'})
export class ViewportSizeDirective implements OnInit, OnDestroy {
  private _visibleSize: ViewportSizeEnum[];
  private _embedded = false;

  constructor(private _viewportSizeService: ViewportSizeService,
              private _templateRef: TemplateRef<any>,
              private _viewContainer: ViewContainerRef,
  ) {
  }

  @Input() set ifViewportSize(sizes: ViewportSizeEnum[]) {
    this._visibleSize = sizes;
  }

  ngOnInit() {
    this._viewportSizeService.size$
      .pipe(
        filter(currentSize => currentSize !== null)
      )
      .subscribe((currentSize: ViewportSizeEnum) => {
        this.onResize(currentSize);
      });
  }

  ngOnDestroy() {
    this._viewportSizeService.size$.unsubscribe();
  }

  onResize(currentSize: ViewportSizeEnum) {
    if (this._visibleSize.includes(currentSize)) {
      if (!this._embedded) {
        this._embedded = true;
        this._viewContainer.createEmbeddedView(this._templateRef);
      }
    } else {
      this._embedded = false;
      this._viewContainer.clear();
    }
  }
}
