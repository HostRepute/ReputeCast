from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.automation_sequence_step import AutomationSequenceStep
  from ..models.automation_step_stats import AutomationStepStats





T = TypeVar("T", bound="AutomationStats")



@_attrs_define
class AutomationStats:
    """ 
        Attributes:
            days (int):
            entered (int):
            in_flow (int):
            sequence (list[AutomationSequenceStep]):
            since (datetime.datetime):
            steps (list[AutomationStepStats]):
     """

    days: int
    entered: int
    in_flow: int
    sequence: list[AutomationSequenceStep]
    since: datetime.datetime
    steps: list[AutomationStepStats]





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_sequence_step import AutomationSequenceStep # noqa: PLC0415
        from ..models.automation_step_stats import AutomationStepStats # noqa: PLC0415
        days = self.days

        entered = self.entered

        in_flow = self.in_flow

        sequence = []
        for sequence_item_data in self.sequence:
            sequence_item = sequence_item_data.to_dict()
            sequence.append(sequence_item)



        since = self.since.isoformat()

        steps = []
        for steps_item_data in self.steps:
            steps_item = steps_item_data.to_dict()
            steps.append(steps_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "entered": entered,
            "inFlow": in_flow,
            "sequence": sequence,
            "since": since,
            "steps": steps,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_sequence_step import AutomationSequenceStep # noqa: PLC0415
        from ..models.automation_step_stats import AutomationStepStats # noqa: PLC0415
        d = dict(src_dict)
        days = d.pop("days")

        entered = d.pop("entered")

        in_flow = d.pop("inFlow")

        sequence = []
        _sequence = d.pop("sequence")
        for sequence_item_data in (_sequence):
            sequence_item = AutomationSequenceStep.from_dict(sequence_item_data)



            sequence.append(sequence_item)


        since = datetime.datetime.fromisoformat(d.pop("since"))




        steps = []
        _steps = d.pop("steps")
        for steps_item_data in (_steps):
            steps_item = AutomationStepStats.from_dict(steps_item_data)



            steps.append(steps_item)


        automation_stats = cls(
            days=days,
            entered=entered,
            in_flow=in_flow,
            sequence=sequence,
            since=since,
            steps=steps,
        )

        return automation_stats

