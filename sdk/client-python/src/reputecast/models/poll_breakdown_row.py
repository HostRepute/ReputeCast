from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.poll_answer_count import PollAnswerCount





T = TypeVar("T", bound="PollBreakdownRow")



@_attrs_define
class PollBreakdownRow:
    """ 
        Attributes:
            answers (int):
            counts (list[PollAnswerCount]):
            key (str):
            label (str):
            percent (float):
            too_few (bool):
            nps (int | Unset):
     """

    answers: int
    counts: list[PollAnswerCount]
    key: str
    label: str
    percent: float
    too_few: bool
    nps: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.poll_answer_count import PollAnswerCount # noqa: PLC0415
        answers = self.answers

        counts = []
        for counts_item_data in self.counts:
            counts_item = counts_item_data.to_dict()
            counts.append(counts_item)



        key = self.key

        label = self.label

        percent = self.percent

        too_few = self.too_few

        nps = self.nps


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answers": answers,
            "counts": counts,
            "key": key,
            "label": label,
            "percent": percent,
            "tooFew": too_few,
        })
        if nps is not UNSET:
            field_dict["nps"] = nps

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.poll_answer_count import PollAnswerCount # noqa: PLC0415
        d = dict(src_dict)
        answers = d.pop("answers")

        counts = []
        _counts = d.pop("counts")
        for counts_item_data in (_counts):
            counts_item = PollAnswerCount.from_dict(counts_item_data)



            counts.append(counts_item)


        key = d.pop("key")

        label = d.pop("label")

        percent = d.pop("percent")

        too_few = d.pop("tooFew")

        nps = d.pop("nps", UNSET)

        poll_breakdown_row = cls(
            answers=answers,
            counts=counts,
            key=key,
            label=label,
            percent=percent,
            too_few=too_few,
            nps=nps,
        )

        return poll_breakdown_row

