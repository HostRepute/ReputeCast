from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.survey_question_type import SurveyQuestionType
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.survey_option import SurveyOption
  from ..models.survey_save import SurveySave





T = TypeVar("T", bound="SurveyQuestion")



@_attrs_define
class SurveyQuestion:
    """ 
        Attributes:
            answers (list[SurveyOption]):
            key (str):
            required (bool):
            save (SurveySave):
            text (str):
            type_ (SurveyQuestionType):
            description (str | Unset):
            faces (int | Unset):
            high (str | Unset):
            low (str | Unset):
            nps (bool | Unset):
            required_text (str | Unset):
     """

    answers: list[SurveyOption]
    key: str
    required: bool
    save: SurveySave
    text: str
    type_: SurveyQuestionType
    description: str | Unset = UNSET
    faces: int | Unset = UNSET
    high: str | Unset = UNSET
    low: str | Unset = UNSET
    nps: bool | Unset = UNSET
    required_text: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.survey_option import SurveyOption # noqa: PLC0415
        from ..models.survey_save import SurveySave # noqa: PLC0415
        answers = []
        for answers_item_data in self.answers:
            answers_item = answers_item_data.to_dict()
            answers.append(answers_item)



        key = self.key

        required = self.required

        save = self.save.to_dict()

        text = self.text

        type_ = self.type_.value

        description = self.description

        faces = self.faces

        high = self.high

        low = self.low

        nps = self.nps

        required_text = self.required_text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answers": answers,
            "key": key,
            "required": required,
            "save": save,
            "text": text,
            "type": type_,
        })
        if description is not UNSET:
            field_dict["description"] = description
        if faces is not UNSET:
            field_dict["faces"] = faces
        if high is not UNSET:
            field_dict["high"] = high
        if low is not UNSET:
            field_dict["low"] = low
        if nps is not UNSET:
            field_dict["nps"] = nps
        if required_text is not UNSET:
            field_dict["requiredText"] = required_text

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.survey_option import SurveyOption # noqa: PLC0415
        from ..models.survey_save import SurveySave # noqa: PLC0415
        d = dict(src_dict)
        answers = []
        _answers = d.pop("answers")
        for answers_item_data in (_answers):
            answers_item = SurveyOption.from_dict(answers_item_data)



            answers.append(answers_item)


        key = d.pop("key")

        required = d.pop("required")

        save = SurveySave.from_dict(d.pop("save"))




        text = d.pop("text")

        type_ = SurveyQuestionType(d.pop("type"))




        description = d.pop("description", UNSET)

        faces = d.pop("faces", UNSET)

        high = d.pop("high", UNSET)

        low = d.pop("low", UNSET)

        nps = d.pop("nps", UNSET)

        required_text = d.pop("requiredText", UNSET)

        survey_question = cls(
            answers=answers,
            key=key,
            required=required,
            save=save,
            text=text,
            type_=type_,
            description=description,
            faces=faces,
            high=high,
            low=low,
            nps=nps,
            required_text=required_text,
        )

        return survey_question

