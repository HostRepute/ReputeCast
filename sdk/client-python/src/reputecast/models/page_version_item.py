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
  from ..models.page_person import PagePerson





T = TypeVar("T", bound="PageVersionItem")



@_attrs_define
class PageVersionItem:
    """ 
        Attributes:
            current (bool):
            label (str): created, autosave, saved, settings, kept, restored, published
            live (bool):
            name (str):
            saved_at (datetime.datetime):
            version (int):
            published_at (datetime.datetime | Unset):
            published_by (str | Unset):
            saved_by (PagePerson | Unset):
     """

    current: bool
    label: str
    live: bool
    name: str
    saved_at: datetime.datetime
    version: int
    published_at: datetime.datetime | Unset = UNSET
    published_by: str | Unset = UNSET
    saved_by: PagePerson | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_person import PagePerson # noqa: PLC0415
        current = self.current

        label = self.label

        live = self.live

        name = self.name

        saved_at = self.saved_at.isoformat()

        version = self.version

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        published_by = self.published_by

        saved_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.saved_by, Unset):
            saved_by = self.saved_by.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "current": current,
            "label": label,
            "live": live,
            "name": name,
            "savedAt": saved_at,
            "version": version,
        })
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if published_by is not UNSET:
            field_dict["publishedBy"] = published_by
        if saved_by is not UNSET:
            field_dict["savedBy"] = saved_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_person import PagePerson # noqa: PLC0415
        d = dict(src_dict)
        current = d.pop("current")

        label = d.pop("label")

        live = d.pop("live")

        name = d.pop("name")

        saved_at = datetime.datetime.fromisoformat(d.pop("savedAt"))




        version = d.pop("version")

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        published_by = d.pop("publishedBy", UNSET)

        _saved_by = d.pop("savedBy", UNSET)
        saved_by: PagePerson | Unset
        if isinstance(_saved_by,  Unset):
            saved_by = UNSET
        else:
            saved_by = PagePerson.from_dict(_saved_by)




        page_version_item = cls(
            current=current,
            label=label,
            live=live,
            name=name,
            saved_at=saved_at,
            version=version,
            published_at=published_at,
            published_by=published_by,
            saved_by=saved_by,
        )

        return page_version_item

