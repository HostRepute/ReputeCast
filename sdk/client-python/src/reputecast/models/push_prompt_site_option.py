from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_prompt_site_option_kind import PushPromptSiteOptionKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="PushPromptSiteOption")



@_attrs_define
class PushPromptSiteOption:
    """ 
        Attributes:
            host (str):
            kind (PushPromptSiteOptionKind):
            origin (str):
            push_site (bool):
            website (bool):
            id (str | Unset):
     """

    host: str
    kind: PushPromptSiteOptionKind
    origin: str
    push_site: bool
    website: bool
    id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        host = self.host

        kind = self.kind.value

        origin = self.origin

        push_site = self.push_site

        website = self.website

        id = self.id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "host": host,
            "kind": kind,
            "origin": origin,
            "pushSite": push_site,
            "website": website,
        })
        if id is not UNSET:
            field_dict["id"] = id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        host = d.pop("host")

        kind = PushPromptSiteOptionKind(d.pop("kind"))




        origin = d.pop("origin")

        push_site = d.pop("pushSite")

        website = d.pop("website")

        id = d.pop("id", UNSET)

        push_prompt_site_option = cls(
            host=host,
            kind=kind,
            origin=origin,
            push_site=push_site,
            website=website,
            id=id,
        )

        return push_prompt_site_option

