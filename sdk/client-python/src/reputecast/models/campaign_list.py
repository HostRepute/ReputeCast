from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.campaign_list_item import CampaignListItem
  from ..models.campaign_status_counts import CampaignStatusCounts





T = TypeVar("T", bound="CampaignList")



@_attrs_define
class CampaignList:
    """ 
        Attributes:
            counts (CampaignStatusCounts):
            items (list[CampaignListItem]):
            page (int):
            page_size (int):
            total (int): Campaigns matching the filter
     """

    counts: CampaignStatusCounts
    items: list[CampaignListItem]
    page: int
    page_size: int
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.campaign_list_item import CampaignListItem # noqa: PLC0415
        from ..models.campaign_status_counts import CampaignStatusCounts # noqa: PLC0415
        counts = self.counts.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        page = self.page

        page_size = self.page_size

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "counts": counts,
            "items": items,
            "page": page,
            "pageSize": page_size,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.campaign_list_item import CampaignListItem # noqa: PLC0415
        from ..models.campaign_status_counts import CampaignStatusCounts # noqa: PLC0415
        d = dict(src_dict)
        counts = CampaignStatusCounts.from_dict(d.pop("counts"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = CampaignListItem.from_dict(items_item_data)



            items.append(items_item)


        page = d.pop("page")

        page_size = d.pop("pageSize")

        total = d.pop("total")

        campaign_list = cls(
            counts=counts,
            items=items,
            page=page,
            page_size=page_size,
            total=total,
        )

        return campaign_list

