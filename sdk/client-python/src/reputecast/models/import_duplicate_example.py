from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_duplicate_example_result_tone import ImportDuplicateExampleResultTone






T = TypeVar("T", bound="ImportDuplicateExample")



@_attrs_define
class ImportDuplicateExample:
    """ 
        Attributes:
            email (str):
            in_repute_cast (str):
            in_source (str):
            result (str):
            result_tone (ImportDuplicateExampleResultTone):
            suppressed (bool):
     """

    email: str
    in_repute_cast: str
    in_source: str
    result: str
    result_tone: ImportDuplicateExampleResultTone
    suppressed: bool





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        in_repute_cast = self.in_repute_cast

        in_source = self.in_source

        result = self.result

        result_tone = self.result_tone.value

        suppressed = self.suppressed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
            "inReputeCast": in_repute_cast,
            "inSource": in_source,
            "result": result,
            "resultTone": result_tone,
            "suppressed": suppressed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        in_repute_cast = d.pop("inReputeCast")

        in_source = d.pop("inSource")

        result = d.pop("result")

        result_tone = ImportDuplicateExampleResultTone(d.pop("resultTone"))




        suppressed = d.pop("suppressed")

        import_duplicate_example = cls(
            email=email,
            in_repute_cast=in_repute_cast,
            in_source=in_source,
            result=result,
            result_tone=result_tone,
            suppressed=suppressed,
        )

        return import_duplicate_example

