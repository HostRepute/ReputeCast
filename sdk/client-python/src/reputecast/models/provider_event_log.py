from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.provider_log_event import ProviderLogEvent





T = TypeVar("T", bound="ProviderEventLog")



@_attrs_define
class ProviderEventLog:
    """ 
        Attributes:
            items (list[ProviderLogEvent]):
            page (int):
            page_size (int):
            retention_days (int):
            total (int):
     """

    items: list[ProviderLogEvent]
    page: int
    page_size: int
    retention_days: int
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.provider_log_event import ProviderLogEvent # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        page = self.page

        page_size = self.page_size

        retention_days = self.retention_days

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "page": page,
            "pageSize": page_size,
            "retentionDays": retention_days,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.provider_log_event import ProviderLogEvent # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = ProviderLogEvent.from_dict(items_item_data)



            items.append(items_item)


        page = d.pop("page")

        page_size = d.pop("pageSize")

        retention_days = d.pop("retentionDays")

        total = d.pop("total")

        provider_event_log = cls(
            items=items,
            page=page,
            page_size=page_size,
            retention_days=retention_days,
            total=total,
        )

        return provider_event_log

