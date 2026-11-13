from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.countdown_settings_language import CountdownSettingsLanguage
from ..models.countdown_settings_size import CountdownSettingsSize
from ..models.countdown_settings_units import CountdownSettingsUnits
from typing import cast

if TYPE_CHECKING:
  from ..models.countdown_ended import CountdownEnded





T = TypeVar("T", bound="CountdownSettings")



@_attrs_define
class CountdownSettings:
    """ 
        Attributes:
            background (str): #RRGGBB Example: #0F172A.
            digits (str): #RRGGBB, at least 4.5:1 on the background Example: #FFFFFF.
            ended (CountdownEnded):
            ends_at (str): The local date and time the countdown ends, YYYY-MM-DDTHH:MM Example: 2026-11-28T23:59.
            labels (str): #RRGGBB Example: #CBD5E1.
            language (CountdownSettingsLanguage): The labels' and the alt text's language: the language version being edited
            size (CountdownSettingsSize):
            time_zone (str): IANA time zone of endsAt Example: America/New_York.
            units (CountdownSettingsUnits):
     """

    background: str
    digits: str
    ended: CountdownEnded
    ends_at: str
    labels: str
    language: CountdownSettingsLanguage
    size: CountdownSettingsSize
    time_zone: str
    units: CountdownSettingsUnits





    def to_dict(self) -> dict[str, Any]:
        from ..models.countdown_ended import CountdownEnded # noqa: PLC0415
        background = self.background

        digits = self.digits

        ended = self.ended.to_dict()

        ends_at = self.ends_at

        labels = self.labels

        language = self.language.value

        size = self.size.value

        time_zone = self.time_zone

        units = self.units.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "background": background,
            "digits": digits,
            "ended": ended,
            "endsAt": ends_at,
            "labels": labels,
            "language": language,
            "size": size,
            "timeZone": time_zone,
            "units": units,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.countdown_ended import CountdownEnded # noqa: PLC0415
        d = dict(src_dict)
        background = d.pop("background")

        digits = d.pop("digits")

        ended = CountdownEnded.from_dict(d.pop("ended"))




        ends_at = d.pop("endsAt")

        labels = d.pop("labels")

        language = CountdownSettingsLanguage(d.pop("language"))




        size = CountdownSettingsSize(d.pop("size"))




        time_zone = d.pop("timeZone")

        units = CountdownSettingsUnits(d.pop("units"))




        countdown_settings = cls(
            background=background,
            digits=digits,
            ended=ended,
            ends_at=ends_at,
            labels=labels,
            language=language,
            size=size,
            time_zone=time_zone,
            units=units,
        )

        return countdown_settings

