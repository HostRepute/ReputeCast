from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.website_post_content_source import WebsitePostContentSource
from ..types import UNSET, Unset






T = TypeVar("T", bound="WebsitePostContent")



@_attrs_define
class WebsitePostContent:
    """ 
        Attributes:
            base_version (int):
            mjml (str):
            force (bool | Unset):
            label (str | Unset):
            source (WebsitePostContentSource | Unset):
     """

    base_version: int
    mjml: str
    force: bool | Unset = UNSET
    label: str | Unset = UNSET
    source: WebsitePostContentSource | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        base_version = self.base_version

        mjml = self.mjml

        force = self.force

        label = self.label

        source: str | Unset = UNSET
        if not isinstance(self.source, Unset):
            source = self.source.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "baseVersion": base_version,
            "mjml": mjml,
        })
        if force is not UNSET:
            field_dict["force"] = force
        if label is not UNSET:
            field_dict["label"] = label
        if source is not UNSET:
            field_dict["source"] = source

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        base_version = d.pop("baseVersion")

        mjml = d.pop("mjml")

        force = d.pop("force", UNSET)

        label = d.pop("label", UNSET)

        _source = d.pop("source", UNSET)
        source: WebsitePostContentSource | Unset
        if isinstance(_source,  Unset):
            source = UNSET
        else:
            source = WebsitePostContentSource(_source)




        website_post_content = cls(
            base_version=base_version,
            mjml=mjml,
            force=force,
            label=label,
            source=source,
        )

        return website_post_content

