from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.push_step_stats import PushStepStats





T = TypeVar("T", bound="PushStepStatsList")



@_attrs_define
class PushStepStatsList:
    """ 
        Attributes:
            steps (list[PushStepStats]):
     """

    steps: list[PushStepStats]





    def to_dict(self) -> dict[str, Any]:
        from ..models.push_step_stats import PushStepStats # noqa: PLC0415
        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "steps": steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.push_step_stats import PushStepStats # noqa: PLC0415
        d = dict(src_dict)
        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = PushStepStats.from_dict(steps_item_data)



            steps.append(steps_item)


        push_step_stats_list = cls(
            steps=steps,
        )

        return push_step_stats_list

