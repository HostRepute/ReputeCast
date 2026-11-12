from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_stream_choice_input_role import ConnectStreamChoiceInputRole
from ..models.connect_stream_choice_input_stream import ConnectStreamChoiceInputStream
from ..types import UNSET, Unset






T = TypeVar("T", bound="ConnectStreamChoiceInput")



@_attrs_define
class ConnectStreamChoiceInput:
    """ 
        Attributes:
            enabled (bool):
            stream (ConnectStreamChoiceInputStream):
            role (ConnectStreamChoiceInputRole | Unset):
            weight (int | Unset):
     """

    enabled: bool
    stream: ConnectStreamChoiceInputStream
    role: ConnectStreamChoiceInputRole | Unset = UNSET
    weight: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        enabled = self.enabled

        stream = self.stream.value

        role: str | Unset = UNSET
        if not isinstance(self.role, Unset):
            role = self.role.value


        weight = self.weight


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
            "stream": stream,
        })
        if role is not UNSET:
            field_dict["role"] = role
        if weight is not UNSET:
            field_dict["weight"] = weight

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        enabled = d.pop("enabled")

        stream = ConnectStreamChoiceInputStream(d.pop("stream"))




        _role = d.pop("role", UNSET)
        role: ConnectStreamChoiceInputRole | Unset
        if isinstance(_role,  Unset):
            role = UNSET
        else:
            role = ConnectStreamChoiceInputRole(_role)




        weight = d.pop("weight", UNSET)

        connect_stream_choice_input = cls(
            enabled=enabled,
            stream=stream,
            role=role,
            weight=weight,
        )

        return connect_stream_choice_input

