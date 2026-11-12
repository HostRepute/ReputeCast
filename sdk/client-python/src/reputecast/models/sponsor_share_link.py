from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SponsorShareLink")



@_attrs_define
class SponsorShareLink:
    """ 
        Attributes:
            created_at (datetime.datetime):
            created_by (str):
            every (bool):
            id (str):
            views (int):
            expires_at (datetime.datetime | Unset):
            last_viewed_at (datetime.datetime | Unset):
            url (str | Unset):
     """

    created_at: datetime.datetime
    created_by: str
    every: bool
    id: str
    views: int
    expires_at: datetime.datetime | Unset = UNSET
    last_viewed_at: datetime.datetime | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        created_by = self.created_by

        every = self.every

        id = self.id

        views = self.views

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        last_viewed_at: str | Unset = UNSET
        if not isinstance(self.last_viewed_at, Unset):
            last_viewed_at = self.last_viewed_at.isoformat()

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "createdBy": created_by,
            "every": every,
            "id": id,
            "views": views,
        })
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if last_viewed_at is not UNSET:
            field_dict["lastViewedAt"] = last_viewed_at
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        every = d.pop("every")

        id = d.pop("id")

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




        url = d.pop("url", UNSET)

        sponsor_share_link = cls(
            created_at=created_at,
            created_by=created_by,
            every=every,
            id=id,
            views=views,
            expires_at=expires_at,
            last_viewed_at=last_viewed_at,
            url=url,
        )

        return sponsor_share_link

