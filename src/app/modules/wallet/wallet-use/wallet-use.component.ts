/*CORE*/
import {Component, Input, OnInit} from '@angular/core';
import {FormArray, FormBuilder, FormGroup, Validators} from '@angular/forms';
import {ActivatedRoute, ParamMap} from '@angular/router';
import {forkJoin, Subscription} from 'rxjs';
import {debounceTime, distinctUntilChanged, filter} from 'rxjs/operators';
/*SERVICES*/
import {WalletService} from '../wallet.service';
import {ToastrService} from '../../toastr/toastr.service';
import {CommonService} from '../../../services/common.service';
/*MODELS*/
import Web3Contract from 'web3/eth/contract';
import {ABIDefinition} from 'web3/eth/abi';
import {Contract} from '../../../models/contract.model';
import {Badge} from '../../../models/badge.model';
import {Address} from '../../../models/address.model';
/*UTILS*/
import {ErcName, InterfaceName} from '../../../utils/enums';
import {ERC_INTERFACE_IDENTIFIERS, INTERFACE_ABI} from '../../../utils/constants';
import {getAbiMethods, makeContractBadges} from '../../../utils/functions';

@Component({
  selector: 'app-wallet-use',
  templateUrl: './wallet-use.component.html',
  styleUrls: ['./wallet-use.component.scss'],
})
export class WalletUseComponent implements OnInit {

  hasData = false;

  @Input('contractData')
  set address([addr, contract]: [Address, Contract]) {
    this.hasData = true;
    this.useContractForm.patchValue({
      contractAddress: addr.address,
    }, {
      emitEvent: false,
    });
    if (contract && contract.abi) {
      this.handleContractData(addr, contract);
    }
  }

  useContractForm: FormGroup = this._fb.group({
    contractAddress: ['', Validators.required],
    contractAmount: ['', []],
    contractABI: ['', []],
    contractFunction: [''],
    functionParameters: this._fb.array([]),
  });

  // Contract stuff
  contract: Web3Contract;
  selectedFunction: ABIDefinition;
  functionResult: any[][];
  functions: ABIDefinition[] = [];

  isProcessing = false;

  contractBadges: Badge[] = [];

  abiTemplates = [ErcName.Erc20, ErcName.Erc721];

  private _subsArr$: Subscription[] = [];

  get functionParameters() {
    return this.useContractForm.get('functionParameters') as FormArray;
  }

  constructor(
    private _walletService: WalletService,
    private _fb: FormBuilder,
    private _toastrService: ToastrService,
    private _activatedRoute: ActivatedRoute,
    private _commonService: CommonService,
  ) {
  }

  ngOnInit() {
    this._subsArr$.push(
      this._activatedRoute.queryParamMap.pipe(
        filter((params: ParamMap) => params.has('address'))
      ).subscribe((params: ParamMap) => {
        const addr = params.get('address');
        if (addr.length === 42) {
          this.useContractForm.patchValue({
            contractAddress: addr
          });
          this.getContractData(addr);
        } else {
          this._toastrService.warning('Contract address is invalid');
        }
      })
    );
    this._subsArr$.push(this.useContractForm.get('contractAddress').valueChanges.pipe(
      debounceTime(500),
      distinctUntilChanged(),
    ).subscribe((val: string) => {
      this.updateContractInfo();
      this.getContractData(val);
    }));
    this._subsArr$.push(this.useContractForm.get('contractABI').valueChanges.pipe(
      debounceTime(500),
      distinctUntilChanged(),
    ).subscribe(val => {
      this.updateContractInfo();
    }));
    this._subsArr$.push(this.useContractForm.get('contractFunction').valueChanges.subscribe(value => {
      this.loadFunction(value);
    }));
  }

  private getContractData(addrHash: string) {
    forkJoin(
      this._commonService.getAddress(addrHash),
      this._commonService.getContract(addrHash),
    ).pipe(
      filter((data: [Address, Contract]) => data[0] && data[1] && data[1].valid && !!data[1].abi.length),
    ).subscribe((data: [Address, Contract]) => {
      this.handleContractData(data[0], data[1]);
    });
  }

  private handleContractData(address: Address, contract: Contract) {
    this.contractBadges = makeContractBadges(address, contract);
    this.useContractForm.patchValue({
      contractABI: JSON.stringify(contract.abi),
    }, {
      emitEvent: false,
    });
    this.initiateContract(contract.abi, address.address);
  }

  private initiateContract(abi: ABIDefinition[], addrHash: string) {
    try {
      this.contract = new this._walletService.w3.eth.Contract(abi, addrHash);
      this.functions = getAbiMethods(this.contract.options.jsonInterface);
    } catch (e) {
      this._toastrService.danger('Can]\'t initiate contract, check entered data');
      return;
    }
  }

  /**
   *
   * @param functionIndex
   */
  loadFunction(functionIndex: number): void {
    this.selectedFunction = null;
    this.functionResult = null;
    this.resetFunctionParameter();
    const func = this.functions[functionIndex];
    this.selectedFunction = func;
    // TODO: IF ANY INPUTS, add a sub formgroup
    // if constant, just show value immediately
    if (func && func.inputs && func.inputs.length) {
      func.inputs.forEach(() => {
        this.addFunctionParameter();
      });
    }
  }

  addFunctionParameter() {
    this.functionParameters.push(this._fb.control(''));
  }

  /**
   *
   * @param func
   * @param params
   */
  callABIFunction(func: ABIDefinition, params: string[]): void {
    this.isProcessing = true;
    let funcABI: string;
    try {
      funcABI = this._walletService.w3.eth.abi.encodeFunctionCall(func, params);
    } catch (err) {
      this._toastrService.danger(err);
      this.isProcessing = false;
      return;
    }
    this._walletService.w3.eth.call({
      to: this.contract.options.address,
      data: funcABI,
    }).then((result: string) => {
      const decoded: object = this._walletService.w3.eth.abi.decodeLog(func.outputs, result, []);
      // This Result object is frikin stupid, it's literaly an empty object that they add fields too
      // convert to something iterable
      const arrR: any[][] = [];
      // let mapR: Map<any,any> = new Map<any,any>();
      // for (let j = 0; j < decoded.__length__; j++){
      //   mapR.push([decoded[0], decoded[1]])
      // }
      Object.keys(decoded).forEach((key) => {
        // mapR[key] = decoded[key];
        if (key.startsWith('__')) {
          return;
        }
        if (!decoded[key].payable || decoded[key].constant) {
          arrR.push([key, decoded[key]]);
        }
      });
      this.functionResult = arrR;
      this.isProcessing = false;
    }).catch(err => {
      this._toastrService.danger(err);
      this.isProcessing = false;
    });
  }

  resetFunctionParameter() {
    while (this.functionParameters.length !== 0) {
      this.functionParameters.removeAt(0);
    }
  }

  reset() {
    this.selectedFunction = null;
  }

  updateContractInfo(): void {
    this.contract = null;
    this.functions = [];
    const addr: string = this.useContractForm.get('contractAddress').value;
    let abi = this.useContractForm.get('contractABI').value;
    if (!addr) {
      return;
    }
    if (addr.length !== 42) {
      this._toastrService.danger('Wrong contract address');
      return;
    }
    if (!abi) {
      return;
    }

    if (abi && abi.length > 0) {
      try {
        abi = JSON.parse(abi);
      } catch (e) {
        this._toastrService.danger('Can\'t parse contract abi');
        return;
      }
      this.initiateContract(abi, addr);
    }
  }

  useContract() {
    const params: string[] = [];

    if (this.selectedFunction.inputs.length) {
      this.functionParameters.controls.forEach(control => {
        params.push(control.value);
      });
    }

    this.callABIFunction(this.selectedFunction, params);
  }

  onAbiTemplateClick(ercName: ErcName) {
    const ABI: ABIDefinition[] = ERC_INTERFACE_IDENTIFIERS[ercName].map((interfaceName: InterfaceName) => INTERFACE_ABI[interfaceName]);
    const addr: string = this.useContractForm.get('contractAddress').value;
    this.useContractForm.patchValue({
      contractABI: JSON.stringify(ABI),
    }, {
      emitEvent: false,
    });
    if (addr.length === 42 && ABI.length) {
      this.initiateContract(ABI, addr);
    }
  }
}
