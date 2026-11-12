from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="AIKeptSummaryView")



@_attrs_define
class AIKeptSummaryView:
    """ 
        Attributes:
            kept_at (datetime.datetime):
            kept_by (str):
            model (str):
            model_name (str):
            numbers_match (bool):
            summary (str):
            unmatched (list[str]):
     """

    kept_at: datetime.datetime
    kept_by: str
    model: str
    model_name: str
    numbers_match: bool
    summary: str
    unmatched: list[str]





    def to_dict(self) -> dict[str, Any]:
        kept_at = self.kept_at.isoformat()

        kept_by = self.kept_by

        model = self.model

        model_name = self.model_name

        numbers_match = self.numbers_match

        summary = self.summary

        unmatched = self.unmatched




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "keptAt": kept_at,
            "keptBy": kept_by,
            "model": model,
            "modelName": model_name,
            "numbersMatch": numbers_match,
            "summary": summary,
            "unmatched": unmatched,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kept_at = datetime.datetime.fromisoformat(d.pop("keptAt"))




        kept_by = d.pop("keptBy")

        model = d.pop("model")

        model_name = d.pop("modelName")

        numbers_match = d.pop("numbersMatch")

        summary = d.pop("summary")

        unmatched = cast(list[str], d.pop("unmatched"))


        ai_kept_summary_view = cls(
            kept_at=kept_at,
            kept_by=kept_by,
            model=model,
            model_name=model_name,
            numbers_match=numbers_match,
            summary=summary,
            unmatched=unmatched,
        )

        return ai_kept_summary_view

