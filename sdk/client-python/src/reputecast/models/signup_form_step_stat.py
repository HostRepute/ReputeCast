from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="SignupFormStepStat")



@_attrs_define
class SignupFormStepStat:
    """ 
        Attributes:
            step (int): 1 to 4
            views (int):
     """

    step: int
    views: int





    def to_dict(self) -> dict[str, Any]:
        step = self.step

        views = self.views


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "step": step,
            "views": views,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        step = d.pop("step")

        views = d.pop("views")

        signup_form_step_stat = cls(
            step=step,
            views=views,
        )

        return signup_form_step_stat

