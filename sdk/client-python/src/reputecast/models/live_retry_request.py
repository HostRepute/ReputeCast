from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from uuid import UUID






T = TypeVar("T", bound="LiveRetryRequest")



@_attrs_define
class LiveRetryRequest:
    """ 
        Attributes:
            provider_id (UUID):
     """

    provider_id: UUID





    def to_dict(self) -> dict[str, Any]:
        provider_id = str(self.provider_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "providerId": provider_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        provider_id = UUID(d.pop("providerId"))




        live_retry_request = cls(
            provider_id=provider_id,
        )

        return live_retry_request

