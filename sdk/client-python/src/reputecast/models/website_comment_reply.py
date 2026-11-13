from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WebsiteCommentReply")



@_attrs_define
class WebsiteCommentReply:
    """ 
        Attributes:
            body (str):
            created_at (datetime.datetime):
            name (str):
            team (bool):
            by (str | Unset):
     """

    body: str
    created_at: datetime.datetime
    name: str
    team: bool
    by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        body = self.body

        created_at = self.created_at.isoformat()

        name = self.name

        team = self.team

        by = self.by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "createdAt": created_at,
            "name": name,
            "team": team,
        })
        if by is not UNSET:
            field_dict["by"] = by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        body = d.pop("body")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        name = d.pop("name")

        team = d.pop("team")

        by = d.pop("by", UNSET)

        website_comment_reply = cls(
            body=body,
            created_at=created_at,
            name=name,
            team=team,
            by=by,
        )

        return website_comment_reply

