from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TransactionalProviderLimit")



@_attrs_define
class TransactionalProviderLimit:
    """ 
        Attributes:
            bytes_ (int):
            provider (str):  Example: SES.
     """

    bytes_: int
    provider: str





    def to_dict(self) -> dict[str, Any]:
        bytes_ = self.bytes_

        provider = self.provider


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bytes": bytes_,
            "provider": provider,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bytes_ = d.pop("bytes")

        provider = d.pop("provider")

        transactional_provider_limit = cls(
            bytes_=bytes_,
            provider=provider,
        )

        return transactional_provider_limit

