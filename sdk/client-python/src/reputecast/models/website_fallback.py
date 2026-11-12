from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebsiteFallback")



@_attrs_define
class WebsiteFallback:
    """ 
        Attributes:
            fallback (str):
            sample (str):
            source (str):
            token (str):
     """

    fallback: str
    sample: str
    source: str
    token: str





    def to_dict(self) -> dict[str, Any]:
        fallback = self.fallback

        sample = self.sample

        source = self.source

        token = self.token


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fallback": fallback,
            "sample": sample,
            "source": source,
            "token": token,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        fallback = d.pop("fallback")

        sample = d.pop("sample")

        source = d.pop("source")

        token = d.pop("token")

        website_fallback = cls(
            fallback=fallback,
            sample=sample,
            source=source,
            token=token,
        )

        return website_fallback

