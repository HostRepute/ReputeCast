from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_suppression import SmsSuppression
  from ..models.sms_suppression_list_counts import SmsSuppressionListCounts
  from ..models.sms_suppression_provider import SmsSuppressionProvider





T = TypeVar("T", bound="SmsSuppressionList")



@_attrs_define
class SmsSuppressionList:
    """ 
        Attributes:
            can_edit (bool):
            counts (SmsSuppressionListCounts):
            filtered_recent (int):
            items (list[SmsSuppression]):
            matching (int):
            page (int):
            page_size (int):
            providers (list[SmsSuppressionProvider]):
            total (int):
            reason (str | Unset):
     """

    can_edit: bool
    counts: SmsSuppressionListCounts
    filtered_recent: int
    items: list[SmsSuppression]
    matching: int
    page: int
    page_size: int
    providers: list[SmsSuppressionProvider]
    total: int
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_suppression import SmsSuppression # noqa: PLC0415
        from ..models.sms_suppression_list_counts import SmsSuppressionListCounts # noqa: PLC0415
        from ..models.sms_suppression_provider import SmsSuppressionProvider # noqa: PLC0415
        can_edit = self.can_edit

        counts = self.counts.to_dict()

        filtered_recent = self.filtered_recent

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        matching = self.matching

        page = self.page

        page_size = self.page_size

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        total = self.total

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "counts": counts,
            "filteredRecent": filtered_recent,
            "items": items,
            "matching": matching,
            "page": page,
            "pageSize": page_size,
            "providers": providers,
            "total": total,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_suppression import SmsSuppression # noqa: PLC0415
        from ..models.sms_suppression_list_counts import SmsSuppressionListCounts # noqa: PLC0415
        from ..models.sms_suppression_provider import SmsSuppressionProvider # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        counts = SmsSuppressionListCounts.from_dict(d.pop("counts"))




        filtered_recent = d.pop("filteredRecent")

        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SmsSuppression.from_dict(items_item_data)



            items.append(items_item)


        matching = d.pop("matching")

        page = d.pop("page")

        page_size = d.pop("pageSize")

        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = SmsSuppressionProvider.from_dict(providers_item_data)



            providers.append(providers_item)


        total = d.pop("total")

        reason = d.pop("reason", UNSET)

        sms_suppression_list = cls(
            can_edit=can_edit,
            counts=counts,
            filtered_recent=filtered_recent,
            items=items,
            matching=matching,
            page=page,
            page_size=page_size,
            providers=providers,
            total=total,
            reason=reason,
        )

        return sms_suppression_list

