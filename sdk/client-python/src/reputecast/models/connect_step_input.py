from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_step_input_step import ConnectStepInputStep






T = TypeVar("T", bound="ConnectStepInput")



@_attrs_define
class ConnectStepInput:
    """ 
        Attributes:
            step (ConnectStepInputStep):
     """

    step: ConnectStepInputStep





    def to_dict(self) -> dict[str, Any]:
        step = self.step.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "step": step,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        step = ConnectStepInputStep(d.pop("step"))




        connect_step_input = cls(
            step=step,
        )

        return connect_step_input

