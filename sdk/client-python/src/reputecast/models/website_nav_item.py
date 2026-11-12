from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_nav_item_kind import WebsiteNavItemKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsiteNavItem")



@_attrs_define
class WebsiteNavItem:
    """ 
        Attributes:
            kind (WebsiteNavItemKind):
            name (str):
            path (str):
            posts (int):
            label (str | Unset):
            ref (str | Unset):
            url (str | Unset):
     """

    kind: WebsiteNavItemKind
    name: str
    path: str
    posts: int
    label: str | Unset = UNSET
    ref: str | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        name = self.name

        path = self.path

        posts = self.posts

        label = self.label

        ref = self.ref

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "name": name,
            "path": path,
            "posts": posts,
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
        kind = WebsiteNavItemKind(d.pop("kind"))




        name = d.pop("name")

        path = d.pop("path")

        posts = d.pop("posts")

        label = d.pop("label", UNSET)

        ref = d.pop("ref", UNSET)

        url = d.pop("url", UNSET)

        website_nav_item = cls(
            kind=kind,
            name=name,
            path=path,
            posts=posts,
            label=label,
            ref=ref,
            url=url,
        )

        return website_nav_item

