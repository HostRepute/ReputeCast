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
  from ..models.report_share_includes import ReportShareIncludes





T = TypeVar("T", bound="ReportShareLink")



@_attrs_define
class ReportShareLink:
    """ 
        Attributes:
            created_at (datetime.datetime):
            created_by_name (str):
            id (str):
            includes (ReportShareIncludes):
            url (str):
            views (int):
            expires_at (datetime.datetime | Unset):
            last_viewed_at (datetime.datetime | Unset):
     """

    created_at: datetime.datetime
    created_by_name: str
    id: str
    includes: ReportShareIncludes
    url: str
    views: int
    expires_at: datetime.datetime | Unset = UNSET
    last_viewed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_share_includes import ReportShareIncludes # noqa: PLC0415
        created_at = self.created_at.isoformat()

        created_by_name = self.created_by_name

        id = self.id

        includes = self.includes.to_dict()

        url = self.url

        views = self.views

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        last_viewed_at: str | Unset = UNSET
        if not isinstance(self.last_viewed_at, Unset):
            last_viewed_at = self.last_viewed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "createdByName": created_by_name,
            "id": id,
            "includes": includes,
            "url": url,
            "views": views,
        })
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if last_viewed_at is not UNSET:
            field_dict["lastViewedAt"] = last_viewed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_share_includes import ReportShareIncludes # noqa: PLC0415
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by_name = d.pop("createdByName")

        id = d.pop("id")

        includes = ReportShareIncludes.from_dict(d.pop("includes"))




        url = d.pop("url")

        views = d.pop("views")

        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        _last_viewed_at = d.pop("lastViewedAt", UNSET)
        last_viewed_at: datetime.datetime | Unset
        if isinstance(_last_viewed_at,  Unset):
            last_viewed_at = UNSET
        else:
            last_viewed_at = datetime.datetime.fromisoformat(_last_viewed_at)




        report_share_link = cls(
            created_at=created_at,
            created_by_name=created_by_name,
            id=id,
            includes=includes,
            url=url,
            views=views,
            expires_at=expires_at,
            last_viewed_at=last_viewed_at,
        )

        return report_share_link

