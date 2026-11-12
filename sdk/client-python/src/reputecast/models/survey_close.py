from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.survey_close_mode import SurveyCloseMode
from ..types import UNSET, Unset






T = TypeVar("T", bound="SurveyClose")



@_attrs_define
class SurveyClose:
    """ 
        Attributes:
            mode (SurveyCloseMode):
            date (str | Unset):
            time (str | Unset):
     """

    mode: SurveyCloseMode
    date: str | Unset = UNSET
    time: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        date = self.date

        time = self.time


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
        })
        if date is not UNSET:
            field_dict["date"] = date
        if time is not UNSET:
            field_dict["time"] = time

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = SurveyCloseMode(d.pop("mode"))




        date = d.pop("date", UNSET)

        time = d.pop("time", UNSET)

        survey_close = cls(
            mode=mode,
            date=date,
            time=time,
        )

        return survey_close

