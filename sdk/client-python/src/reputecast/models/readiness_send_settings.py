from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.readiness_schedule import ReadinessSchedule
  from ..models.readiness_thresholds import ReadinessThresholds





T = TypeVar("T", bound="ReadinessSendSettings")



@_attrs_define
class ReadinessSendSettings:
    """ 
        Attributes:
            include_holdout (bool | Unset):
            schedule (ReadinessSchedule | Unset):
            thresholds (ReadinessThresholds | Unset):
     """

    include_holdout: bool | Unset = UNSET
    schedule: ReadinessSchedule | Unset = UNSET
    thresholds: ReadinessThresholds | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.readiness_schedule import ReadinessSchedule # noqa: PLC0415
        from ..models.readiness_thresholds import ReadinessThresholds # noqa: PLC0415
        include_holdout = self.include_holdout

        schedule: dict[str, Any] | Unset = UNSET
        if not isinstance(self.schedule, Unset):
            schedule = self.schedule.to_dict()

        thresholds: dict[str, Any] | Unset = UNSET
        if not isinstance(self.thresholds, Unset):
            thresholds = self.thresholds.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if include_holdout is not UNSET:
            field_dict["includeHoldout"] = include_holdout
        if schedule is not UNSET:
            field_dict["schedule"] = schedule
        if thresholds is not UNSET:
            field_dict["thresholds"] = thresholds

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.readiness_schedule import ReadinessSchedule # noqa: PLC0415
        from ..models.readiness_thresholds import ReadinessThresholds # noqa: PLC0415
        d = dict(src_dict)
        include_holdout = d.pop("includeHoldout", UNSET)

        _schedule = d.pop("schedule", UNSET)
        schedule: ReadinessSchedule | Unset
        if isinstance(_schedule,  Unset):
            schedule = UNSET
        else:
            schedule = ReadinessSchedule.from_dict(_schedule)




        _thresholds = d.pop("thresholds", UNSET)
        thresholds: ReadinessThresholds | Unset
        if isinstance(_thresholds,  Unset):
            thresholds = UNSET
        else:
            thresholds = ReadinessThresholds.from_dict(_thresholds)




        readiness_send_settings = cls(
            include_holdout=include_holdout,
            schedule=schedule,
            thresholds=thresholds,
        )

        return readiness_send_settings

