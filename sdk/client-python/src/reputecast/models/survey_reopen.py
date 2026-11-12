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





T = TypeVar("T", bound="SurveyReopen")



@_attrs_define
class SurveyReopen:
    """ 
        Attributes:
            close (SurveyClose | Unset):
     """

    close: SurveyClose | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.survey_close import SurveyClose # noqa: PLC0415
        close: dict[str, Any] | Unset = UNSET
        if not isinstance(self.close, Unset):
            close = self.close.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if close is not UNSET:
            field_dict["close"] = close

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.survey_close import SurveyClose # noqa: PLC0415
        d = dict(src_dict)
        _close = d.pop("close", UNSET)
        close: SurveyClose | Unset
        if isinstance(_close,  Unset):
            close = UNSET
        else:
            close = SurveyClose.from_dict(_close)




        survey_reopen = cls(
            close=close,
        )

        return survey_reopen

