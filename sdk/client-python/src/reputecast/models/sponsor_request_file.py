from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_request_file_kind import SponsorRequestFileKind






T = TypeVar("T", bound="SponsorRequestFile")



@_attrs_define
class SponsorRequestFile:
    """ 
        Attributes:
            kind (SponsorRequestFileKind):
            name (str):
            size_bytes (int):
            url (str):
     """

    kind: SponsorRequestFileKind
    name: str
    size_bytes: int
    url: str





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        name = self.name

        size_bytes = self.size_bytes

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "name": name,
            "sizeBytes": size_bytes,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = SponsorRequestFileKind(d.pop("kind"))




        name = d.pop("name")

        size_bytes = d.pop("sizeBytes")

        url = d.pop("url")

        sponsor_request_file = cls(
            kind=kind,
            name=name,
            size_bytes=size_bytes,
            url=url,
        )

        return sponsor_request_file

