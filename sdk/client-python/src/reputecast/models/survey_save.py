from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.survey_save_to import SurveySaveTo
from ..types import UNSET, Unset






T = TypeVar("T", bound="SurveySave")



@_attrs_define
class SurveySave:
    """ 
        Attributes:
            to (SurveySaveTo):
            field (str | Unset):
     """

    to: SurveySaveTo
    field: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        to = self.to.value

        field = self.field


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "to": to,
        })
        if field is not UNSET:
            field_dict["field"] = field

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        to = SurveySaveTo(d.pop("to"))




        field = d.pop("field", UNSET)

        survey_save = cls(
            to=to,
            field=field,
        )

        return survey_save

