from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sms_step_stats import SmsStepStats





T = TypeVar("T", bound="SmsStepStatsList")



@_attrs_define
class SmsStepStatsList:
    """ 
        Attributes:
            days (int):
            steps (list[SmsStepStats]):
     """

    days: int
    steps: list[SmsStepStats]





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_step_stats import SmsStepStats # noqa: PLC0415
        days = self.days

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "steps": steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_step_stats import SmsStepStats # noqa: PLC0415
        d = dict(src_dict)
        days = d.pop("days")

        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = SmsStepStats.from_dict(steps_item_data)



            steps.append(steps_item)


        sms_step_stats_list = cls(
            days=days,
            steps=steps,
        )

        return sms_step_stats_list

