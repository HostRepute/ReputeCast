from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AIModelOption")



@_attrs_define
class AIModelOption:
    """ 
        Attributes:
            id (str):
            image_input (bool):
            input_ (float): Dollars per million input tokens
            known (bool): On ReputeCast's price list
            name (str):
            output (float): Dollars per million output tokens
     """

    id: str
    image_input: bool
    input_: float
    known: bool
    name: str
    output: float





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        image_input = self.image_input

        input_ = self.input_

        known = self.known

        name = self.name

        output = self.output


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "imageInput": image_input,
            "input": input_,
            "known": known,
            "name": name,
            "output": output,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        image_input = d.pop("imageInput")

        input_ = d.pop("input")

        known = d.pop("known")

        name = d.pop("name")

        output = d.pop("output")

        ai_model_option = cls(
            id=id,
            image_input=image_input,
            input_=input_,
            known=known,
            name=name,
            output=output,
        )

        return ai_model_option

