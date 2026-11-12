from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_in_doubt_request_action import LiveInDoubtRequestAction
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="LiveInDoubtRequest")



@_attrs_define
class LiveInDoubtRequest:
    """ 
        Attributes:
            action (LiveInDoubtRequestAction):
            provider_id (UUID | Unset):
     """

    action: LiveInDoubtRequestAction
    provider_id: UUID | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        provider_id: str | Unset = UNSET
        if not isinstance(self.provider_id, Unset):
            provider_id = str(self.provider_id)


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
        })
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = LiveInDoubtRequestAction(d.pop("action"))




        _provider_id = d.pop("providerId", UNSET)
        provider_id: UUID | Unset
        if isinstance(_provider_id,  Unset):
            provider_id = UNSET
        else:
            provider_id = UUID(_provider_id)




        live_in_doubt_request = cls(
            action=action,
            provider_id=provider_id,
        )

        return live_in_doubt_request

