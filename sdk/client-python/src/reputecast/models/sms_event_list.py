from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_event import SmsEvent
  from ..models.sms_opt_out_sync_row import SmsOptOutSyncRow
  from ..models.sms_suppression_provider import SmsSuppressionProvider





T = TypeVar("T", bound="SmsEventList")



@_attrs_define
class SmsEventList:
    """ 
        Attributes:
            items (list[SmsEvent]):
            month (datetime.datetime):
            opt_out_sync (list[SmsOptOutSyncRow]):
            page (int):
            page_size (int):
            providers (list[SmsSuppressionProvider]):
            suppressed (int):
            this_month (int):
            total (int):
     """

    items: list[SmsEvent]
    month: datetime.datetime
    opt_out_sync: list[SmsOptOutSyncRow]
    page: int
    page_size: int
    providers: list[SmsSuppressionProvider]
    suppressed: int
    this_month: int
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_event import SmsEvent # noqa: PLC0415
        from ..models.sms_opt_out_sync_row import SmsOptOutSyncRow # noqa: PLC0415
        from ..models.sms_suppression_provider import SmsSuppressionProvider # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        month = self.month.isoformat()

        opt_out_sync = []
        for opt_out_sync_item_data in self.opt_out_sync:
            opt_out_sync_item = opt_out_sync_item_data.to_dict()
            opt_out_sync.append(opt_out_sync_item)



        page = self.page

        page_size = self.page_size

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        suppressed = self.suppressed

        this_month = self.this_month

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "month": month,
            "optOutSync": opt_out_sync,
            "page": page,
            "pageSize": page_size,
            "providers": providers,
            "suppressed": suppressed,
            "thisMonth": this_month,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_event import SmsEvent # noqa: PLC0415
        from ..models.sms_opt_out_sync_row import SmsOptOutSyncRow # noqa: PLC0415
        from ..models.sms_suppression_provider import SmsSuppressionProvider # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SmsEvent.from_dict(items_item_data)



            items.append(items_item)


        month = datetime.datetime.fromisoformat(d.pop("month"))




        opt_out_sync = []
        _opt_out_sync = d.pop("optOutSync")
        for opt_out_sync_item_data in (_opt_out_sync):
            opt_out_sync_item = SmsOptOutSyncRow.from_dict(opt_out_sync_item_data)



            opt_out_sync.append(opt_out_sync_item)


        page = d.pop("page")

        page_size = d.pop("pageSize")

        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = SmsSuppressionProvider.from_dict(providers_item_data)



            providers.append(providers_item)


        suppressed = d.pop("suppressed")

        this_month = d.pop("thisMonth")

        total = d.pop("total")

        sms_event_list = cls(
            items=items,
            month=month,
            opt_out_sync=opt_out_sync,
            page=page,
            page_size=page_size,
            providers=providers,
            suppressed=suppressed,
            this_month=this_month,
            total=total,
        )

        return sms_event_list

