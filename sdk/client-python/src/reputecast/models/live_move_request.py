from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from uuid import UUID






T = TypeVar("T", bound="LiveMoveRequest")



@_attrs_define
class LiveMoveRequest:
    """ 
        Attributes:
            from_provider_id (UUID):
            to_provider_id (UUID):
     """

    from_provider_id: UUID
    to_provider_id: UUID





    def to_dict(self) -> dict[str, Any]:
        from_provider_id = str(self.from_provider_id)

        to_provider_id = str(self.to_provider_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fromProviderId": from_provider_id,
            "toProviderId": to_provider_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        from_provider_id = UUID(d.pop("fromProviderId"))




        to_provider_id = UUID(d.pop("toProviderId"))




        live_move_request = cls(
            from_provider_id=from_provider_id,
            to_provider_id=to_provider_id,
        )

        return live_move_request

