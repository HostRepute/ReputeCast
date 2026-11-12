from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.suppression_counts_view import SuppressionCountsView
  from ..models.suppression_filter_option import SuppressionFilterOption
  from ..models.suppression_item import SuppressionItem
  from ..models.suppression_provider_sync import SuppressionProviderSync





T = TypeVar("T", bound="SuppressionList")



@_attrs_define
class SuppressionList:
    """ 
        Attributes:
            counts (SuppressionCountsView):
            items (list[SuppressionItem]):
            matching (int): Addresses matching the filter; at most 10,000 unless exact
            page (int):
            page_size (int):
            sources (list[SuppressionFilterOption]):
            sync (list[SuppressionProviderSync]):
            total (int): Rows matching, groups counted once; at most 10,000 unless exact
            next_cursor (str | Unset):
            prev_cursor (str | Unset):
            total_capped (bool | Unset):
     """

    counts: SuppressionCountsView
    items: list[SuppressionItem]
    matching: int
    page: int
    page_size: int
    sources: list[SuppressionFilterOption]
    sync: list[SuppressionProviderSync]
    total: int
    next_cursor: str | Unset = UNSET
    prev_cursor: str | Unset = UNSET
    total_capped: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.suppression_counts_view import SuppressionCountsView # noqa: PLC0415
        from ..models.suppression_filter_option import SuppressionFilterOption # noqa: PLC0415
        from ..models.suppression_item import SuppressionItem # noqa: PLC0415
        from ..models.suppression_provider_sync import SuppressionProviderSync # noqa: PLC0415
        counts = self.counts.to_dict()

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        matching = self.matching

        page = self.page

        page_size = self.page_size

        sources = []
        for sources_item_data in self.sources:
            sources_item = sources_item_data.to_dict()
            sources.append(sources_item)



        sync = []
        for sync_item_data in self.sync:
            sync_item = sync_item_data.to_dict()
            sync.append(sync_item)



        total = self.total

        next_cursor = self.next_cursor

        prev_cursor = self.prev_cursor

        total_capped = self.total_capped


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "counts": counts,
            "items": items,
            "matching": matching,
            "page": page,
            "pageSize": page_size,
            "sources": sources,
            "sync": sync,
            "total": total,
        })
        if next_cursor is not UNSET:
            field_dict["nextCursor"] = next_cursor
        if prev_cursor is not UNSET:
            field_dict["prevCursor"] = prev_cursor
        if total_capped is not UNSET:
            field_dict["totalCapped"] = total_capped

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.suppression_counts_view import SuppressionCountsView # noqa: PLC0415
        from ..models.suppression_filter_option import SuppressionFilterOption # noqa: PLC0415
        from ..models.suppression_item import SuppressionItem # noqa: PLC0415
        from ..models.suppression_provider_sync import SuppressionProviderSync # noqa: PLC0415
        d = dict(src_dict)
        counts = SuppressionCountsView.from_dict(d.pop("counts"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = SuppressionItem.from_dict(items_item_data)



            items.append(items_item)


        matching = d.pop("matching")

        page = d.pop("page")

        page_size = d.pop("pageSize")

        sources = []
        _sources = d.pop("sources")
        for sources_item_data in (_sources):
            sources_item = SuppressionFilterOption.from_dict(sources_item_data)



            sources.append(sources_item)


        sync = []
        _sync = d.pop("sync")
        for sync_item_data in (_sync):
            sync_item = SuppressionProviderSync.from_dict(sync_item_data)



            sync.append(sync_item)


        total = d.pop("total")

        next_cursor = d.pop("nextCursor", UNSET)

        prev_cursor = d.pop("prevCursor", UNSET)

        total_capped = d.pop("totalCapped", UNSET)

        suppression_list = cls(
            counts=counts,
            items=items,
            matching=matching,
            page=page,
            page_size=page_size,
            sources=sources,
            sync=sync,
            total=total,
            next_cursor=next_cursor,
            prev_cursor=prev_cursor,
            total_capped=total_capped,
        )

        return suppression_list

