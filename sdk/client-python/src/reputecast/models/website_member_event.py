from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_member_event_kind import WebsiteMemberEventKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WebsiteMemberEvent")



@_attrs_define
class WebsiteMemberEvent:
    """ 
        Attributes:
            at (datetime.datetime):
            id (str):
            kind (WebsiteMemberEventKind):
            actor (str | Unset):
            detail (str | Unset):
            method (str | Unset):
            same_browser (bool | Unset):
     """

    at: datetime.datetime
    id: str
    kind: WebsiteMemberEventKind
    actor: str | Unset = UNSET
    detail: str | Unset = UNSET
    method: str | Unset = UNSET
    same_browser: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        id = self.id

        kind = self.kind.value

        actor = self.actor

        detail = self.detail

        method = self.method

        same_browser = self.same_browser


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "id": id,
            "kind": kind,
        })
        if actor is not UNSET:
            field_dict["actor"] = actor
        if detail is not UNSET:
            field_dict["detail"] = detail
        if method is not UNSET:
            field_dict["method"] = method
        if same_browser is not UNSET:
            field_dict["sameBrowser"] = same_browser

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        id = d.pop("id")

        kind = WebsiteMemberEventKind(d.pop("kind"))




        actor = d.pop("actor", UNSET)

        detail = d.pop("detail", UNSET)

        method = d.pop("method", UNSET)

        same_browser = d.pop("sameBrowser", UNSET)

        website_member_event = cls(
            at=at,
            id=id,
            kind=kind,
            actor=actor,
            detail=detail,
            method=method,
            same_browser=same_browser,
        )

        return website_member_event

