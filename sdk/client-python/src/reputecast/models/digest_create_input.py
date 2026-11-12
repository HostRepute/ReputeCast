from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_create_input_source import DigestCreateInputSource
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="DigestCreateInput")



@_attrs_define
class DigestCreateInput:
    """ 
        Attributes:
            source (DigestCreateInputSource):
            feed_url (str | Unset):
            name (str | Unset):
            site_id (UUID | Unset):
     """

    source: DigestCreateInputSource
    feed_url: str | Unset = UNSET
    name: str | Unset = UNSET
    site_id: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        source = self.source.value

        feed_url = self.feed_url

        name = self.name

        site_id: str | Unset = UNSET
        if not isinstance(self.site_id, Unset):
            site_id = str(self.site_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "source": source,
        })
        if feed_url is not UNSET:
            field_dict["feedUrl"] = feed_url
        if name is not UNSET:
            field_dict["name"] = name
        if site_id is not UNSET:
            field_dict["siteId"] = site_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        source = DigestCreateInputSource(d.pop("source"))




        feed_url = d.pop("feedUrl", UNSET)

        name = d.pop("name", UNSET)

        _site_id = d.pop("siteId", UNSET)
        site_id: UUID | Unset
        if isinstance(_site_id,  Unset):
            site_id = UNSET
        else:
            site_id = UUID(_site_id)




        digest_create_input = cls(
            source=source,
            feed_url=feed_url,
            name=name,
            site_id=site_id,
        )

        return digest_create_input

