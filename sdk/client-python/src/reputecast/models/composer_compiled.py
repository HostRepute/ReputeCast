from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.mjml_problem import MJMLProblem





T = TypeVar("T", bound="ComposerCompiled")



@_attrs_define
class ComposerCompiled:
    """ 
        Attributes:
            duration_ms (int):
            errors (list[MJMLProblem]):
            html (str):
            size_bytes (int):
            warnings (list[MJMLProblem]):
     """

    duration_ms: int
    errors: list[MJMLProblem]
    html: str
    size_bytes: int
    warnings: list[MJMLProblem]





    def to_dict(self) -> dict[str, Any]:
        from ..models.mjml_problem import MJMLProblem # noqa: PLC0415
        duration_ms = self.duration_ms

        errors = []
        for errors_item_data in self.errors:
            errors_item = errors_item_data.to_dict()
            errors.append(errors_item)



        html = self.html

        size_bytes = self.size_bytes

        warnings = []
        for warnings_item_data in self.warnings:
            warnings_item = warnings_item_data.to_dict()
            warnings.append(warnings_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "durationMs": duration_ms,
            "errors": errors,
            "html": html,
            "sizeBytes": size_bytes,
            "warnings": warnings,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.mjml_problem import MJMLProblem # noqa: PLC0415
        d = dict(src_dict)
        duration_ms = d.pop("durationMs")

        errors = []
        _errors = d.pop("errors")
        for errors_item_data in (_errors):
            errors_item = MJMLProblem.from_dict(errors_item_data)



            errors.append(errors_item)


        html = d.pop("html")

        size_bytes = d.pop("sizeBytes")

        warnings = []
        _warnings = d.pop("warnings")
        for warnings_item_data in (_warnings):
            warnings_item = MJMLProblem.from_dict(warnings_item_data)



            warnings.append(warnings_item)


        composer_compiled = cls(
            duration_ms=duration_ms,
            errors=errors,
            html=html,
            size_bytes=size_bytes,
            warnings=warnings,
        )

        return composer_compiled

