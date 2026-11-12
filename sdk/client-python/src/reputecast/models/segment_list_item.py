from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.segment_list_item_mode import SegmentListItemMode
from ..models.segment_list_item_refresh import SegmentListItemRefresh
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SegmentListItem")



@_attrs_define
class SegmentListItem:
    """ 
        Attributes:
            id (str):
            mode (SegmentListItemMode):
            name (str):
            refresh (SegmentListItemRefresh):
            updated_at (datetime.datetime):
            count (int | Unset): Last known count
            counted_at (datetime.datetime | Unset):
     """

    id: str
    mode: SegmentListItemMode
    name: str
    refresh: SegmentListItemRefresh
    updated_at: datetime.datetime
    count: int | Unset = UNSET
    counted_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        mode = self.mode.value

        name = self.name

        refresh = self.refresh.value

        updated_at = self.updated_at.isoformat()

        count = self.count

        counted_at: str | Unset = UNSET
        if not isinstance(self.counted_at, Unset):
            counted_at = self.counted_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "mode": mode,
            "name": name,
            "refresh": refresh,
            "updatedAt": updated_at,
        })
        if count is not UNSET:
            field_dict["count"] = count
        if counted_at is not UNSET:
            field_dict["countedAt"] = counted_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        mode = SegmentListItemMode(d.pop("mode"))




        name = d.pop("name")

        refresh = SegmentListItemRefresh(d.pop("refresh"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        count = d.pop("count", UNSET)

        _counted_at = d.pop("countedAt", UNSET)
        counted_at: datetime.datetime | Unset
        if isinstance(_counted_at,  Unset):
            counted_at = UNSET
        else:
            counted_at = datetime.datetime.fromisoformat(_counted_at)




        segment_list_item = cls(
            id=id,
            mode=mode,
            name=name,
            refresh=refresh,
            updated_at=updated_at,
            count=count,
            counted_at=counted_at,
        )

        return segment_list_item

