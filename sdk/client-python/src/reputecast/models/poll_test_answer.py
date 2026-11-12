from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="PollTestAnswer")



@_attrs_define
class PollTestAnswer:
    """ 
        Attributes:
            answer (int):
            at (datetime.datetime):
            block_id (str):
            email (str):
            poll_id (str):
     """

    answer: int
    at: datetime.datetime
    block_id: str
    email: str
    poll_id: str





    def to_dict(self) -> dict[str, Any]:
        answer = self.answer

        at = self.at.isoformat()

        block_id = self.block_id

        email = self.email

        poll_id = self.poll_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answer": answer,
            "at": at,
            "blockId": block_id,
            "email": email,
            "pollId": poll_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        answer = d.pop("answer")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        block_id = d.pop("blockId")

        email = d.pop("email")

        poll_id = d.pop("pollId")

        poll_test_answer = cls(
            answer=answer,
            at=at,
            block_id=block_id,
            email=email,
            poll_id=poll_id,
        )

        return poll_test_answer

