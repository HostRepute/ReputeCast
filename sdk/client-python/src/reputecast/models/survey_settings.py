from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.survey_close import SurveyClose





T = TypeVar("T", bound="SurveySettings")



@_attrs_define
class SurveySettings:
    """ 
        Attributes:
            accept_non_contacts (bool):
            close (SurveyClose):
            one_response (bool):
            thanks (str | Unset):
            timezone (str | Unset):
     """

    accept_non_contacts: bool
    close: SurveyClose
    one_response: bool
    thanks: str | Unset = UNSET
    timezone: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.survey_close import SurveyClose # noqa: PLC0415
        accept_non_contacts = self.accept_non_contacts

        close = self.close.to_dict()

        one_response = self.one_response

        thanks = self.thanks

        timezone = self.timezone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "acceptNonContacts": accept_non_contacts,
            "close": close,
            "oneResponse": one_response,
        })
        if thanks is not UNSET:
            field_dict["thanks"] = thanks
        if timezone is not UNSET:
            field_dict["timezone"] = timezone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.survey_close import SurveyClose # noqa: PLC0415
        d = dict(src_dict)
        accept_non_contacts = d.pop("acceptNonContacts")

        close = SurveyClose.from_dict(d.pop("close"))




        one_response = d.pop("oneResponse")

        thanks = d.pop("thanks", UNSET)

        timezone = d.pop("timezone", UNSET)

        survey_settings = cls(
            accept_non_contacts=accept_non_contacts,
            close=close,
            one_response=one_response,
            thanks=thanks,
            timezone=timezone,
        )

        return survey_settings

