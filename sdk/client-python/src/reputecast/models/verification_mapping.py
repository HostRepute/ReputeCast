from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_mapping_result import VerificationMappingResult
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="VerificationMapping")



@_attrs_define
class VerificationMapping:
    """ 
        Attributes:
            codes (list[str]):
            result (VerificationMappingResult):
            reason (str | Unset):
            tag (str | Unset):
     """

    codes: list[str]
    result: VerificationMappingResult
    reason: str | Unset = UNSET
    tag: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        codes = self.codes



        result = self.result.value

        reason = self.reason

        tag = self.tag


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "codes": codes,
            "result": result,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason
        if tag is not UNSET:
            field_dict["tag"] = tag

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        codes = cast(list[str], d.pop("codes"))


        result = VerificationMappingResult(d.pop("result"))




        reason = d.pop("reason", UNSET)

        tag = d.pop("tag", UNSET)

        verification_mapping = cls(
            codes=codes,
            result=result,
            reason=reason,
            tag=tag,
        )

        return verification_mapping

