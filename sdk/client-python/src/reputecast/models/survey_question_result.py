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
  from ..models.poll_nps import PollNPS





T = TypeVar("T", bound="SurveyQuestionResult")



@_attrs_define
class SurveyQuestionResult:
    """ 
        Attributes:
            answered (int):
            counts (list[PollAnswerCount]):
            key (str):
            number (int):
            required (bool):
            text (str):
            type_ (str):
            nps (PollNPS | Unset):
            saved_to (str | Unset):
     """

    answered: int
    counts: list[PollAnswerCount]
    key: str
    number: int
    required: bool
    text: str
    type_: str
    nps: PollNPS | Unset = UNSET
    saved_to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.poll_answer_count import PollAnswerCount # noqa: PLC0415
        from ..models.poll_nps import PollNPS # noqa: PLC0415
        answered = self.answered

        counts = []
        for counts_item_data in self.counts:
            counts_item = counts_item_data.to_dict()
            counts.append(counts_item)



        key = self.key

        number = self.number

        required = self.required

        text = self.text

        type_ = self.type_

        nps: dict[str, Any] | Unset = UNSET
        if not isinstance(self.nps, Unset):
            nps = self.nps.to_dict()

        saved_to = self.saved_to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answered": answered,
            "counts": counts,
            "key": key,
            "number": number,
            "required": required,
            "text": text,
            "type": type_,
        })
        if nps is not UNSET:
            field_dict["nps"] = nps
        if saved_to is not UNSET:
            field_dict["savedTo"] = saved_to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.poll_answer_count import PollAnswerCount # noqa: PLC0415
        from ..models.poll_nps import PollNPS # noqa: PLC0415
        d = dict(src_dict)
        answered = d.pop("answered")

        counts = []
        _counts = d.pop("counts")
        for counts_item_data in (_counts):
            counts_item = PollAnswerCount.from_dict(counts_item_data)



            counts.append(counts_item)


        key = d.pop("key")

        number = d.pop("number")

        required = d.pop("required")

        text = d.pop("text")

        type_ = d.pop("type")

        _nps = d.pop("nps", UNSET)
        nps: PollNPS | Unset
        if isinstance(_nps,  Unset):
            nps = UNSET
        else:
            nps = PollNPS.from_dict(_nps)




        saved_to = d.pop("savedTo", UNSET)

        survey_question_result = cls(
            answered=answered,
            counts=counts,
            key=key,
            number=number,
            required=required,
            text=text,
            type_=type_,
            nps=nps,
            saved_to=saved_to,
        )

        return survey_question_result

