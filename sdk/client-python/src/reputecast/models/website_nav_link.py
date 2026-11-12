from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_nav_link_kind import WebsiteNavLinkKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsiteNavLink")



@_attrs_define
class WebsiteNavLink:
    """ 
        Attributes:
            kind (WebsiteNavLinkKind):
            label (str | Unset):
            ref (str | Unset):
            url (str | Unset):
     """

    kind: WebsiteNavLinkKind
    label: str | Unset = UNSET
    ref: str | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        label = self.label

        ref = self.ref

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if label is not UNSET:
            field_dict["label"] = label
        if ref is not UNSET:
            field_dict["ref"] = ref
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = WebsiteNavLinkKind(d.pop("kind"))




        label = d.pop("label", UNSET)

        ref = d.pop("ref", UNSET)

        url = d.pop("url", UNSET)

        website_nav_link = cls(
            kind=kind,
            label=label,
            ref=ref,
            url=url,
        )

        return website_nav_link

