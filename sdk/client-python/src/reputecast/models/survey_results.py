from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.survey_question_result import SurveyQuestionResult





T = TypeVar("T", bound="SurveyResults")



@_attrs_define
class SurveyResults:
    """ 
        Attributes:
            completed (int):
            completion_percent (float):
            non_contacts (int):
            poll_only (int):
            questions (list[SurveyQuestionResult]):
            recipients (int):
            started (int):
     """

    completed: int
    completion_percent: float
    non_contacts: int
    poll_only: int
    questions: list[SurveyQuestionResult]
    recipients: int
    started: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.survey_question_result import SurveyQuestionResult # noqa: PLC0415
        completed = self.completed

        completion_percent = self.completion_percent

        non_contacts = self.non_contacts

        poll_only = self.poll_only

        questions = []
        for questions_item_data in self.questions:
            questions_item = questions_item_data.to_dict()
            questions.append(questions_item)



        recipients = self.recipients

        started = self.started


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "completed": completed,
            "completionPercent": completion_percent,
            "nonContacts": non_contacts,
            "pollOnly": poll_only,
            "questions": questions,
            "recipients": recipients,
            "started": started,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.survey_question_result import SurveyQuestionResult # noqa: PLC0415
        d = dict(src_dict)
        completed = d.pop("completed")

        completion_percent = d.pop("completionPercent")

        non_contacts = d.pop("nonContacts")

        poll_only = d.pop("pollOnly")

        questions = []
        _questions = d.pop("questions")
        for questions_item_data in (_questions):
            questions_item = SurveyQuestionResult.from_dict(questions_item_data)



            questions.append(questions_item)


        recipients = d.pop("recipients")

        started = d.pop("started")

        survey_results = cls(
            completed=completed,
            completion_percent=completion_percent,
            non_contacts=non_contacts,
            poll_only=poll_only,
            questions=questions,
            recipients=recipients,
            started=started,
        )

        return survey_results

