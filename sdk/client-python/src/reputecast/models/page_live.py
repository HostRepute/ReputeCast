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





T = TypeVar("T", bound="PageLive")



@_attrs_define
class PageLive:
    """ 
        Attributes:
            path (str):
            slug (str):
            url (str):
            version (int):
            published_at (datetime.datetime | Unset):
            published_by (PagePerson | Unset):
     """

    path: str
    slug: str
    url: str
    version: int
    published_at: datetime.datetime | Unset = UNSET
    published_by: PagePerson | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.page_person import PagePerson # noqa: PLC0415
        path = self.path

        slug = self.slug

        url = self.url

        version = self.version

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        published_by: dict[str, Any] | Unset = UNSET
        if not isinstance(self.published_by, Unset):
            published_by = self.published_by.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "path": path,
            "slug": slug,
            "url": url,
            "version": version,
        })
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if published_by is not UNSET:
            field_dict["publishedBy"] = published_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.page_person import PagePerson # noqa: PLC0415
        d = dict(src_dict)
        path = d.pop("path")

        slug = d.pop("slug")

        url = d.pop("url")

        version = d.pop("version")

        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        _published_by = d.pop("publishedBy", UNSET)
        published_by: PagePerson | Unset
        if isinstance(_published_by,  Unset):
            published_by = UNSET
        else:
            published_by = PagePerson.from_dict(_published_by)




        page_live = cls(
            path=path,
            slug=slug,
            url=url,
            version=version,
            published_at=published_at,
            published_by=published_by,
        )

        return page_live

