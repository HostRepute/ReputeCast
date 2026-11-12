from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.contact_filter import ContactFilter





T = TypeVar("T", bound="ContactSavedView")



@_attrs_define
class ContactSavedView:
    """ 
        Attributes:
            columns (list[str]):
            count (int):
            filters (ContactFilter):
            id (str):
            name (str):
            shared (bool):
            counted_at (datetime.datetime | Unset):
     """

    columns: list[str]
    count: int
    filters: ContactFilter
    id: str
    name: str
    shared: bool
    counted_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        columns = self.columns



        count = self.count

        filters = self.filters.to_dict()

        id = self.id

        name = self.name

        shared = self.shared

        counted_at: str | Unset = UNSET
        if not isinstance(self.counted_at, Unset):
            counted_at = self.counted_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "columns": columns,
            "count": count,
            "filters": filters,
            "id": id,
            "name": name,
            "shared": shared,
        })
        if counted_at is not UNSET:
            field_dict["countedAt"] = counted_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_filter import ContactFilter # noqa: PLC0415
        d = dict(src_dict)
        columns = cast(list[str], d.pop("columns"))


        count = d.pop("count")

        filters = ContactFilter.from_dict(d.pop("filters"))




        id = d.pop("id")

        name = d.pop("name")

        shared = d.pop("shared")

        _counted_at = d.pop("countedAt", UNSET)
        counted_at: datetime.datetime | Unset
        if isinstance(_counted_at,  Unset):
            counted_at = UNSET
        else:
            counted_at = datetime.datetime.fromisoformat(_counted_at)




        contact_saved_view = cls(
            columns=columns,
            count=count,
            filters=filters,
            id=id,
            name=name,
            shared=shared,
            counted_at=counted_at,
        )

        return contact_saved_view

