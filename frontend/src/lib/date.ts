/**
 * Fuso do Brasil usado para exibir e comparar datas no frontend.
 * Garante que "hoje" e intervalos de data sigam o horário de Brasília.
 */
const BRAZIL_TZ = 'America/Sao_Paulo'

/**
 * Retorna a data no fuso do Brasil no formato YYYY-MM-DD.
 * Use em vez de date.toISOString().slice(0, 10), que usa UTC e pode trocar o dia no Brasil.
 */
export function toBrazilYYYYMMDD(d: Date): string {
  return d.toLocaleDateString('en-CA', { timeZone: BRAZIL_TZ })
}
